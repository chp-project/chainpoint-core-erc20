package abci

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chainpoint/tendermint/libs/log"
	core_types "github.com/chainpoint/tendermint/rpc/core/types"
	"github.com/chp-project/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/tendermint/rpc/client"
	"github.com/chp-project/chainpoint-core/go-abci-service/util"
)

// RPC : hold abstract http client for mocking purposes
type RPC struct {
	client *client.HTTP
	logger log.Logger
}

// NewRPCClient : Creates a new client connected to a tendermint instance at web socket "tendermintRPC"
func NewRPCClient(tendermintRPC types.TendermintConfig, logger log.Logger) (rpc *RPC) {
	return &RPC{
		client: client.NewHTTP(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket"),
		logger: logger,
	}
}

//LogError : log rpc errors
func (rpc *RPC) LogError(err error) error {
	if err != nil {
		rpc.logger.Error(fmt.Sprintf("Error: %s", err.Error()))
	}
	return err
}

// BroadcastTx : Synchronously broadcasts a transaction to the local Tendermint node
func (rpc *RPC) BroadcastTx(txType string, data string, version int64, time int64, stackID string, privateKey *ecdsa.PrivateKey) (core_types.ResultBroadcastTx, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID}
	result, err := rpc.client.BroadcastTxSync([]byte(util.EncodeTxWithKey(tx, privateKey)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastTx{}, err
	}
	return *result, nil
}

// BroadcastTx : Synchronously broadcasts a transaction to the local Tendermint node
func (rpc *RPC) BroadcastTxWithMeta(txType string, data string, version int64, time int64, stackID string, meta string, privateKey *ecdsa.PrivateKey) (core_types.ResultBroadcastTx, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID, Meta: meta}
	result, err := rpc.client.BroadcastTxSync([]byte(util.EncodeTxWithKey(tx, privateKey)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastTx{}, err
	}
	return *result, nil
}

func (rpc *RPC) BroadcastMsg(tx types.Tx) (core_types.ResultBroadcastMsg, error) {
	result, err := rpc.client.BroadcastMsgAsync([]byte(util.EncodeTx(tx)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastMsg{}, err
	}
	return *result, nil
}

// BroadcastTxCommit : Synchronously broadcasts a transaction to the local Tendermint node THIS IS BLOCKING
func (rpc *RPC) BroadcastTxCommit(txType string, data string, version int64, time int64, stackID string, privateKey *ecdsa.PrivateKey) (core_types.ResultBroadcastTxCommit, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID}
	result, err := rpc.client.BroadcastTxCommit([]byte(util.EncodeTxWithKey(tx, privateKey)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastTxCommit{}, err
	}
	return *result, nil
}

// GetStatus retrieves status of our node. Can't use RPC because remote_ip has buggy encoding.
func (rpc *RPC) GetStatus() (core_types.ResultStatus, error) {
	if rpc == nil {
		return core_types.ResultStatus{}, errors.New("rpc failure")
	}
	status, err := rpc.client.Status()
	if rpc.LogError(err) != nil {
		return core_types.ResultStatus{}, err
	}
	return *status, err
}

// GetNetInfo retrieves known peer information. Can't use RPC because remote_ip has buggy encoding.
func (rpc *RPC) GetNetInfo() (core_types.ResultNetInfo, error) {
	if rpc == nil {
		return core_types.ResultNetInfo{}, errors.New("rpc failure")
	}
	netInfo, err := rpc.client.NetInfo()
	if rpc.LogError(err) != nil {
		return core_types.ResultNetInfo{}, err
	}
	return *netInfo, err
}

//GetTxByInt : Retrieves a tx by its unique integer ID (txInt)
func (rpc *RPC) GetTxByInt(txInt int64) (core_types.ResultTxSearch, error) {
	txResult, err := rpc.client.TxSearch(fmt.Sprintf("TxInt=%d", txInt), false, 1, 1)
	if rpc.LogError(err) != nil {
		return core_types.ResultTxSearch{}, err
	}
	return *txResult, err
}

// GetAbciInfo retrieves custom ABCI status struct detailing the state of our application
func (rpc *RPC) GetAbciInfo() (types.AnchorState, error) {
	resp, err := rpc.client.ABCIInfo()
	if err != nil {
		return types.AnchorState{}, err
	}
	var anchorState types.AnchorState
	util.LogError(json.Unmarshal([]byte(resp.Response.Data), &anchorState))
	return anchorState, nil
}

//GetValidators : retrieves list of validators at a particular block height
func (rpc *RPC) GetValidators(height int64) (core_types.ResultValidators, error) {
	resp, err := rpc.client.Validators(&height)
	if err != nil {
		return core_types.ResultValidators{}, err
	}
	return *resp, nil
}
