package abci

import (
	"fmt"
	"sort"

	"github.com/chainpoint/tendermint/types"

	"github.com/chainpoint/tendermint/p2p"
	core_types "github.com/chainpoint/tendermint/rpc/core/types"
	"github.com/chp-project/chainpoint-core/go-abci-service/util"
)

// ElectLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func (app *AnchorApplication) ElectLeader(numLeaders int) (isLeader bool, leaderID []string) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	netInfo, err := app.rpc.GetNetInfo()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	blockHash := status.SyncInfo.LatestBlockHash.String()
	app.logger.Info(fmt.Sprintf("Blockhash Seed: %s", blockHash))
	return determineLeader(numLeaders, status, netInfo, blockHash)
}

// ElectValidator : elect a slice of validators as a leader and return whether we're the leader
func (app *AnchorApplication) ElectValidator(numLeaders int) (isLeader bool, leaderID []string) {
	validators, err := app.rpc.GetValidators(app.state.Height)
	if app.LogError(err) != nil {
		return false, []string{}
	}
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	blockHash := status.SyncInfo.LatestBlockHash.String()
	app.logger.Info(fmt.Sprintf("Blockhash Seed: %s", blockHash))
	return determineValidatorLeader(numLeaders, status, validators, blockHash, app.config.FilePV.GetAddress().String())
}

func determineValidatorLeader(numLeaders int, status core_types.ResultStatus, validators core_types.ResultValidators, seed string, address string) (isLeader bool, leaderIDs []string) {
	leaders := make([]types.Validator, 0)
	validatorList := GetSortedValidatorList(validators)
	validatorLength := len(validatorList)
	index := util.GetSeededRandInt([]byte(seed), validatorLength)    //seed the first time
	if err := util.RotateLeft(validatorList[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
		util.LogError(err)
		return false, []string{}
	}
	if numLeaders <= validatorLength {
		leaders = validatorList[0:numLeaders]
	} else {
		leaders = validatorList[0:1]
	}
	leaderStrings := make([]string, 0)
	iAmLeader := false
	for _, leader := range leaders {
		leaderID := leader.Address.String()
		leaderStrings = append(leaderStrings, leaderID)
		if leaderID == address && !status.SyncInfo.CatchingUp {
			iAmLeader = true
		}
	}
	return iAmLeader, leaderStrings
}

// GetSortedValidatorList : collate and deterministically sort validator list
func GetSortedValidatorList(validators core_types.ResultValidators) []types.Validator {
	validatorList := make([]types.Validator, 0)
	for _, val := range validators.Validators {
		validatorList = append(validatorList, *val)
	}
	sort.Slice(validatorList[:], func(i, j int) bool {
		return validatorList[i].Address.String() > validatorList[j].Address.String()
	})
	return validatorList
}

// GetPeers : get list of all peers
func (app *AnchorApplication) GetPeers() []core_types.Peer {
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	status, err = app.rpc.GetStatus()
	netInfo, err2 = app.rpc.GetNetInfo()

	if app.LogError(err) != nil || util.LogError(err2) != nil {
		return []core_types.Peer{}
	}

	peers := GetSortedPeerList(status, netInfo)
	return peers
}

//GetSortedPeerList : returns sorted list of peers including self
func GetSortedPeerList(status core_types.ResultStatus, netInfo core_types.ResultNetInfo) []core_types.Peer {
	peers := netInfo.Peers
	nodeArray := make([]core_types.Peer, 0)
	for i := 0; i < len(peers); i++ {
		peers[i].RemoteIP = util.DetermineIP(peers[i])
		nodeArray = append(nodeArray, peers[i])
	}
	selfPeer := core_types.Peer{
		NodeInfo:         status.NodeInfo,
		IsOutbound:       false,
		ConnectionStatus: p2p.ConnectionStatus{},
		RemoteIP:         "127.0.0.1",
	}
	nodeArray = append(nodeArray, selfPeer)
	sort.Slice(nodeArray[:], func(i, j int) bool {
		return nodeArray[i].NodeInfo.ID() > nodeArray[j].NodeInfo.ID()
	})
	return nodeArray
}

// determineLeader accepts current node status and a peer array, then finds a leader based on the latest blockhash
func determineLeader(numLeaders int, status core_types.ResultStatus, netInfo core_types.ResultNetInfo, seed string) (isLeader bool, leaderIDs []string) {
	currentNodeID := status.NodeInfo.ID()
	if len(netInfo.Peers) > 0 {
		nodeArray := GetSortedPeerList(status, netInfo)
		index := util.GetSeededRandInt([]byte(seed), len(nodeArray)) //seed the first time
		if err := util.RotateLeft(nodeArray[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
			util.LogError(err)
			return false, []string{}
		}
		leaders := make([]core_types.Peer, 0)
		if numLeaders <= len(nodeArray) {
			leaders = nodeArray[0:numLeaders]
		} else {
			leaders = nodeArray[0:1]
		}
		leaderStrings := make([]string, 0)
		iAmLeader := false
		for _, leader := range leaders {
			leaderStrings = append(leaderStrings, string(leader.NodeInfo.ID()))
			if leader.NodeInfo.ID() == currentNodeID && !status.SyncInfo.CatchingUp {
				iAmLeader = true
			}
		}
		return iAmLeader, leaderStrings
	}
	return true, []string{string(currentNodeID)}
}
