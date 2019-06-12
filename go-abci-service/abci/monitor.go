package abci

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	beacon "github.com/chainpoint/go-nist-beacon"

	"github.com/streadway/amqp"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// ConsumeBtcTxMsg : Consumes a btctx RMQ message to initiate monitoring on all nodes
func (app *AnchorApplication) ConsumeBtcTxMsg(msgBytes []byte) error {
	var btcTxObj types.BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return util.LogError(err)
	}
	app.state.LatestBtcTx = btcTxObj.BtcTxID // Update app state with txID so we can broadcast BTC-A
	stateObj := types.BtcTxProofState{
		AnchorBtcAggID: btcTxObj.AnchorBtcAggID,
		BtcTxID:        btcTxObj.BtcTxID,
		BtcTxState: types.BtcTxOpsState{
			Ops: []types.ProofLineItem{
				types.ProofLineItem{
					Left: btcTxObj.BtcTxBody[:strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)],
				},
				types.ProofLineItem{
					Right: btcTxObj.BtcTxBody[strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)+len(btcTxObj.AnchorBtcAggRoot):],
				},
				types.ProofLineItem{
					Op: "sha-256-x2",
				},
			},
		},
	}
	dataJSON, err := json.Marshal(stateObj)
	if util.LogError(err) != nil {
		return err
	}
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btctx", dataJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID})
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.btcmon", "", txIDBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}

// ConsumeBtcMonMsg : consumes a btc mon message and issues a BTC-Confirm transaction along with completing btc proof generation
func (app *AnchorApplication) ConsumeBtcMonMsg(msg amqp.Delivery) error {
	var anchoringCoreID string
	var hash []byte
	var btcMonObj types.BtcMonMsg
	json.Unmarshal(msg.Body, &btcMonObj)
	// Get the CoreID that originally published the anchor TX using the btc tx ID we tagged it with
	txResult, err := app.rpc.client.TxSearch(fmt.Sprintf("BTCTX=%s", btcMonObj.BtcTxID), false, 1, 25)
	util.LoggerError(app.logger, err)
	for _, tx := range txResult.Txs {
		decoded, err := util.DecodeTx(tx.Tx)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		anchoringCoreID = decoded.CoreID
	}
	if len(anchoringCoreID) == 0 {
		app.logger.Error(fmt.Sprintf("Anchor: Cannot retrieve BTCTX-tagged transaction for btc tx: %s", btcMonObj.BtcTxID))
	}
	// Broadcast the confirmation message with metadata
	result, err := app.rpc.BroadcastTxWithMeta("BTC-C", btcMonObj.BtcHeadRoot, 2, time.Now().Unix(), app.ID, anchoringCoreID+"|"+btcMonObj.BtcTxID, &app.config.ECPrivateKey)
	time.Sleep(1 * time.Minute) // wait until it hits the mempool
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Error(fmt.Sprintf("Anchor: Another core has probably already committed a BTCC tx: %s", err.Error()))
		txResult, err := app.rpc.GetTxByInt(app.state.LatestBtccTxInt)
		if util.LogError(err) != nil && len(txResult.Txs) > 0 {
			hash = txResult.Txs[0].Hash
		} else {
			return err
		}
	} else {
		hash = result.Hash
	}
	var btccStateObj types.BtccStateObj
	btccStateObj.BtcTxID = btcMonObj.BtcTxID
	btccStateObj.BtcHeadHeight = btcMonObj.BtcHeadHeight
	btccStateObj.BtcHeadState.Ops = make([]types.ProofLineItem, 0)
	for _, p := range btcMonObj.Path {
		if p.Left != "" {
			btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Left: string(p.Left)})
		}
		if p.Right != "" {
			btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Right: string(p.Right)})
		}
		btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Op: "sha-256-x2"})
	}
	baseURI := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "https://tendermint.chainpoint.org")
	uri := strings.ToLower(fmt.Sprintf("%s/calendar/%x/data", baseURI, hash))
	btccStateObj.BtcHeadState.Anchor = types.AnchorObj{
		AnchorID: strconv.FormatInt(btcMonObj.BtcHeadHeight, 10),
		Uris:     []string{uri},
	}
	stateObjBytes, err := json.Marshal(btccStateObj)

	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btcmon", stateObjBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	msg.Ack(false)
	return nil
}

func (app *AnchorApplication) processMessage(msg amqp.Delivery) error {
	switch msg.Type {
	case "btctx":
		time.Sleep(30 * time.Second)
		_, err := app.rpc.BroadcastTx("BTC-M", string(msg.Body), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		msg.Ack(false)
		break
	case "btcmon":
		err := app.ConsumeBtcMonMsg(msg)
		util.LogError(err)
		break
	case "reward":
		break
	default:
		msg.Ack(false)
	}
	return nil
}

// ReceiveCalRMQ : Continually consume the calendar work queue and
// process any resulting messages from the tx and monitor services
func (app *AnchorApplication) ReceiveCalRMQ() error {
	var session rabbitmq.Session
	var err error
	endConsume := false
	for {
		session, err = rabbitmq.ConnectAndConsume(app.config.RabbitmqURI, "work.cal")
		if err != nil {
			rabbitmq.LogError(err, "failed to dial for work.cal queue")
			time.Sleep(5 * time.Second)
			continue
		}
		for {
			select {
			case err = <-session.Notify:
				if endConsume {
					return err
				}
				time.Sleep(5 * time.Second)
				break //reconnect
			case msg := <-session.Msgs:
				if len(msg.Body) > 0 {
					go app.processMessage(msg)
				}
			}
		}
	}
}

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		status, err := app.rpc.GetStatus()
		if util.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if app.ID == "" {
			app.ID = string(status.NodeInfo.ID())
		}
		if status.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
		}
		time.Sleep(30 * time.Second)
	}
}

//KeyMonitor : updates active ECDSA public keys from all accessible peers
func (app *AnchorApplication) KeyMonitor() {
	selfStatusURL := fmt.Sprintf("%s/status", app.config.APIURI)
	response, err := http.Get(selfStatusURL)
	if util.LoggerError(app.logger, err) != nil {
		return
	}
	contents, err := ioutil.ReadAll(response.Body)
	if util.LoggerError(app.logger, err) != nil {
		return
	}
	var apiStatus types.CoreAPIStatus
	err = json.Unmarshal(contents, &apiStatus)
	if util.LoggerError(app.logger, err) != nil {
		return
	}
	app.JWK = apiStatus.Jwk
	jwkJson, err := json.Marshal(apiStatus.Jwk)
	if util.LoggerError(app.logger, err) != nil {
		return
	}
	_, err = app.rpc.BroadcastTx("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
	if util.LoggerError(app.logger, err) != nil {
		return
	}
}

// NistBeaconMonitor : elects a leader to poll and gossip NIST. Called every minute by ABCI.commit
func (app *AnchorApplication) NistBeaconMonitor() {
	time.Sleep(15 * time.Second) //sleep after commit for a few seconds
	if leader, leaders := app.ElectValidator(1); leader && app.state.ChainSynced {
		app.logger.Info(fmt.Sprintf("NIST: Elected as leader. Leaders: %v", leaders))
		nistRecord, err := beacon.LastRecord()
		if util.LogError(err) != nil {
			app.logger.Error("Unable to obtain new NIST beacon value")
			return
		}
		_, err = app.rpc.BroadcastTx("NIST", nistRecord.ChainpointFormat(), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
		if util.LogError(err) != nil {
			app.logger.Debug(fmt.Sprintf("Failed to gossip NIST beacon value of %s", nistRecord.ChainpointFormat()))
		}
	}
}

//MintMonitor : efficiently monitor for new minting and gossip that block to other cores
func (app *AnchorApplication) MintMonitor() {
	if leader, _ := app.ElectValidator(1); leader && app.state.ChainSynced {
		lastNodeMintedAt, err := app.ethClient.GetNodeLastMintedAt()
		if util.LogError(err) != nil {
			app.logger.Error("Unable to obtain new NodeLastMintedAt value")
			return
		}
		if lastNodeMintedAt.Int64() != 0 && lastNodeMintedAt.Int64() >= app.state.LastNodeMintedAtBlock+MINT_EPOCH {
			app.logger.Info("Mint success, sending Node MINT tx")
			_, err = app.rpc.BroadcastTx("NODE-MINT", strconv.FormatInt(lastNodeMintedAt.Int64(), 10), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if err != nil {
				app.logger.Debug("Failed to gossip Node MINT for LastNodeMintedAtBlock gossip")
			}
		}
		lastCoreMintedAt, err := app.ethClient.GetCoreLastMintedAt()
		if util.LogError(err) != nil {
			app.logger.Error("Unable to obtain new CoreLastMintedAt value")
			return
		}
		if lastCoreMintedAt.Int64() != 0 && lastCoreMintedAt.Int64() >= app.state.LastCoreMintedAtBlock+MINT_EPOCH {
			app.logger.Info("Mint success, sending Core MINT tx")
			_, err = app.rpc.BroadcastTx("CORE-MINT", strconv.FormatInt(lastCoreMintedAt.Int64(), 10), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if err != nil {
				app.logger.Debug("Failed to gossip Core MINT for LastNodeMintedAtBlock gossip")
			}
		}
	}
}
