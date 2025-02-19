package calendar

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chp-project/chainpoint-core/go-abci-service/rabbitmq"

	"github.com/chp-project/chainpoint-core/go-abci-service/util"
	"github.com/stretchr/testify/assert"

	types2 "github.com/chainpoint/tendermint/types"

	core_types "github.com/chainpoint/tendermint/rpc/core/types"

	"github.com/chp-project/chainpoint-core/go-abci-service/types"
)

func TestEmptyCalTreeGeneration(t *testing.T) {
	cal := NewCalendar("")
	calAggregation := cal.GenerateCalendarTree([]types.Aggregation{})
	if calAggregation.CalRoot != "" {
		t.Errorf("Calendar Aggregation Tree should be empty but obtained %s instead", calAggregation.CalRoot)
	}
}

func TestFullCalTreeGeneration(t *testing.T) {
	cal := NewCalendar("")
	aggregationItems := []types.Aggregation{
		types.Aggregation{AggID: "f4c5445a-49ca-11e9-b3cf-0242ac190005", AggRoot: "89b4f6c13d489cd5e5bd557234cf00d4daeedc2dd50a1f18e525296e5abd1399", ProofData: []types.ProofData{
			types.ProofData{HashID: "d297b590-49ca-11e9-83c4-01bcf13503c2", Hash: "c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2", Proof: []types.ProofLineItem{
				types.ProofLineItem{Left: "core_id:d297b590-49ca-11e9-83c4-01bcf13503c2", Right: "", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"},
				types.ProofLineItem{Left: "", Right: "5c44f9f7764f808729edb87f7857f0cefe3f02c45c41670fe476237adbcc66f0", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}},
			types.ProofData{HashID: "d46808c0-49ca-11e9-83c4-01077bdc855a", Hash: "c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2", Proof: []types.ProofLineItem{
				types.ProofLineItem{Left: "core_id:d46808c0-49ca-11e9-83c4-01077bdc855a", Right: "", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"},
				types.ProofLineItem{Left: "04a284b232924661fe811f18cabe5a789f2fe40ac5c684aecc57545abd72de1b", Right: "", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}}}},
		types.Aggregation{AggID: "f4c549c3-49ca-11e9-b3cf-0242ac190005", AggRoot: "e05d2a67a74278dd4bca5866f4c5a5e66ba7a2217205959bfb27fc0e727aa7d0", ProofData: []types.ProofData{
			types.ProofData{HashID: "d333a770-49ca-11e9-83c4-01ca0c374194", Hash: "c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2", Proof: []types.ProofLineItem{
				types.ProofLineItem{Left: "core_id:d333a770-49ca-11e9-83c4-01ca0c374194", Right: "", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"},
				types.ProofLineItem{Left: "", Right: "79f25ff04782f833e9cecedf738c0694ef8662d36f7dab4496c16bca4da36bbd", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}},
			types.ProofData{HashID: "d3cdeba0-49ca-11e9-83c4-01407c7798ae", Hash: "c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2", Proof: []types.ProofLineItem{
				types.ProofLineItem{Left: "core_id:d3cdeba0-49ca-11e9-83c4-01407c7798ae", Right: "", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"},
				types.ProofLineItem{Left: "99d85ca25f295c5f4d841d706d9c2ccc4ac9528edc09a2a993d9c6553ace5be2", Right: "", Op: ""},
				types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}}}}}

	calAggregation := cal.GenerateCalendarTree(aggregationItems)

	calAggregationResult := types.CalAgg{CalRoot: "80e157fd9029660d2900391b003ee0017671431e5e37fb46dc252877e13355ec", ProofData: []types.CalProofData{
		types.CalProofData{AggID: "f4c5445a-49ca-11e9-b3cf-0242ac190005", Proof: []types.ProofLineItem{
			types.ProofLineItem{Left: "", Right: "e05d2a67a74278dd4bca5866f4c5a5e66ba7a2217205959bfb27fc0e727aa7d0", Op: ""},
			types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}},
		types.CalProofData{AggID: "f4c549c3-49ca-11e9-b3cf-0242ac190005", Proof: []types.ProofLineItem{
			types.ProofLineItem{Left: "89b4f6c13d489cd5e5bd557234cf00d4daeedc2dd50a1f18e525296e5abd1399", Right: "", Op: ""},
			types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}}}}
	if calAggregation.CalRoot != calAggregation.CalRoot {
		t.Errorf("CalRoots don't match; expected %s, got %s\n", calAggregationResult.CalRoot, calAggregation.CalRoot)
	}
	if len(calAggregation.ProofData) != len(calAggregationResult.ProofData) {
		t.Errorf("Partial CalProofs don't match; expected %#v, got %#v\n", calAggregationResult.ProofData, calAggregation.ProofData)
	}
}

func TestEmptyAnchorTreeGeneration(t *testing.T) {
	cal := NewCalendar("")
	resultTx := []core_types.ResultTx{}
	anchorAggregation := cal.AggregateAnchorTx(resultTx)
	if anchorAggregation.AnchorBtcAggRoot != "" {
		t.Errorf("Aggregation Tree should be empty\n ")
	}
}

func TestFullAnchorTreeGeneration(t *testing.T) {
	cal := NewCalendar("")
	aggregationItems := []core_types.ResultTx{
		core_types.ResultTx{
			Tx: types2.Tx{0x65, 0x79, 0x4a, 0x30, 0x65, 0x58, 0x42, 0x6c,
				0x49, 0x6a, 0x6f, 0x69, 0x51, 0x30, 0x46, 0x4d,
				0x49, 0x69, 0x77, 0x69, 0x5a, 0x47, 0x46, 0x30,
				0x59, 0x53, 0x49, 0x36, 0x49, 0x6a, 0x59, 0x77,
				0x59, 0x54, 0x67, 0x77, 0x4e, 0x54, 0x42, 0x69,
				0x4e, 0x6d, 0x59, 0x31, 0x5a, 0x6a, 0x5a, 0x6c,
				0x4f, 0x57, 0x56, 0x6b, 0x4e, 0x54, 0x49, 0x34,
				0x4d, 0x44, 0x67, 0x33, 0x4e, 0x54, 0x42, 0x6d,
				0x4e, 0x44, 0x55, 0x34, 0x4d, 0x57, 0x45, 0x33,
				0x4e, 0x6d, 0x5a, 0x6d, 0x4d, 0x7a, 0x6b, 0x30,
				0x5a, 0x44, 0x42, 0x69, 0x5a, 0x47, 0x56, 0x6a,
				0x4e, 0x57, 0x4d, 0x33, 0x4d, 0x57, 0x45, 0x33,
				0x4f, 0x44, 0x41, 0x79, 0x4e, 0x6a, 0x5a, 0x6b,
				0x4e, 0x44, 0x68, 0x69, 0x5a, 0x6a, 0x4d, 0x34,
				0x4e, 0x7a, 0x51, 0x69, 0x4c, 0x43, 0x4a, 0x32,
				0x5a, 0x58, 0x4a, 0x7a, 0x61, 0x57, 0x39, 0x75,
				0x49, 0x6a, 0x6f, 0x79, 0x4c, 0x43, 0x4a, 0x30,
				0x61, 0x57, 0x31, 0x6c, 0x49, 0x6a, 0x6f, 0x78,
				0x4e, 0x54, 0x55, 0x79, 0x4f, 0x54, 0x51, 0x34,
				0x4d, 0x7a, 0x41, 0x31, 0x66, 0x51, 0x3d, 0x3d},
		},
		core_types.ResultTx{
			Tx: types2.Tx{0x65, 0x79, 0x4a, 0x30, 0x65, 0x58, 0x42, 0x6c,
				0x49, 0x6a, 0x6f, 0x69, 0x51, 0x30, 0x46, 0x4d,
				0x49, 0x69, 0x77, 0x69, 0x5a, 0x47, 0x46, 0x30,
				0x59, 0x53, 0x49, 0x36, 0x49, 0x6a, 0x55, 0x30,
				0x59, 0x54, 0x5a, 0x68, 0x5a, 0x57, 0x46, 0x6b,
				0x59, 0x6a, 0x52, 0x6b, 0x5a, 0x54, 0x67, 0x35,
				0x4e, 0x7a, 0x51, 0x35, 0x4d, 0x7a, 0x6b, 0x34,
				0x59, 0x6a, 0x6c, 0x69, 0x4f, 0x54, 0x6b, 0x32,
				0x59, 0x57, 0x56, 0x6c, 0x4d, 0x44, 0x56, 0x6d,
				0x4e, 0x6a, 0x51, 0x35, 0x4f, 0x44, 0x51, 0x34,
				0x4d, 0x44, 0x4a, 0x6a, 0x4e, 0x44, 0x6c, 0x6b,
				0x4d, 0x57, 0x52, 0x69, 0x59, 0x57, 0x46, 0x6d,
				0x4e, 0x47, 0x46, 0x68, 0x4d, 0x7a, 0x63, 0x35,
				0x5a, 0x57, 0x51, 0x33, 0x59, 0x54, 0x55, 0x7a,
				0x59, 0x6a, 0x45, 0x69, 0x4c, 0x43, 0x4a, 0x32,
				0x5a, 0x58, 0x4a, 0x7a, 0x61, 0x57, 0x39, 0x75,
				0x49, 0x6a, 0x6f, 0x79, 0x4c, 0x43, 0x4a, 0x30,
				0x61, 0x57, 0x31, 0x6c, 0x49, 0x6a, 0x6f, 0x78,
				0x4e, 0x54, 0x55, 0x79, 0x4f, 0x54, 0x51, 0x34,
				0x4d, 0x7a, 0x59, 0x31, 0x66, 0x51, 0x3d, 0x3d},
		},
		core_types.ResultTx{
			Tx: types2.Tx{0x65, 0x79, 0x4a, 0x30, 0x65, 0x58, 0x42, 0x6c,
				0x49, 0x6a, 0x6f, 0x69, 0x51, 0x30, 0x46, 0x4d,
				0x49, 0x69, 0x77, 0x69, 0x5a, 0x47, 0x46, 0x30,
				0x59, 0x53, 0x49, 0x36, 0x49, 0x6d, 0x45, 0x79,
				0x4d, 0x54, 0x45, 0x35, 0x5a, 0x57, 0x49, 0x35,
				0x4e, 0x7a, 0x6b, 0x34, 0x59, 0x54, 0x41, 0x33,
				0x4d, 0x32, 0x51, 0x30, 0x4e, 0x44, 0x67, 0x7a,
				0x59, 0x7a, 0x51, 0x34, 0x5a, 0x44, 0x46, 0x6a,
				0x59, 0x57, 0x49, 0x79, 0x4e, 0x57, 0x4e, 0x6a,
				0x4d, 0x6a, 0x4d, 0x78, 0x4f, 0x57, 0x51, 0x78,
				0x4d, 0x7a, 0x64, 0x6b, 0x4e, 0x7a, 0x63, 0x79,
				0x5a, 0x6d, 0x45, 0x32, 0x5a, 0x6a, 0x49, 0x77,
				0x5a, 0x6d, 0x59, 0x30, 0x4e, 0x6a, 0x41, 0x34,
				0x4d, 0x47, 0x4d, 0x32, 0x4e, 0x54, 0x59, 0x78,
				0x59, 0x57, 0x55, 0x69, 0x4c, 0x43, 0x4a, 0x32,
				0x5a, 0x58, 0x4a, 0x7a, 0x61, 0x57, 0x39, 0x75,
				0x49, 0x6a, 0x6f, 0x79, 0x4c, 0x43, 0x4a, 0x30,
				0x61, 0x57, 0x31, 0x6c, 0x49, 0x6a, 0x6f, 0x78,
				0x4e, 0x54, 0x55, 0x79, 0x4f, 0x54, 0x51, 0x34,
				0x4e, 0x44, 0x49, 0x31, 0x66, 0x51, 0x3d, 0x3d},
		},
		core_types.ResultTx{
			Tx: types2.Tx{0x65, 0x79, 0x4a, 0x30, 0x65, 0x58, 0x42, 0x6c,
				0x49, 0x6a, 0x6f, 0x69, 0x51, 0x30, 0x46, 0x4d,
				0x49, 0x69, 0x77, 0x69, 0x5a, 0x47, 0x46, 0x30,
				0x59, 0x53, 0x49, 0x36, 0x49, 0x6a, 0x41, 0x78,
				0x5a, 0x54, 0x6b, 0x79, 0x4f, 0x54, 0x41, 0x31,
				0x4e, 0x7a, 0x41, 0x77, 0x4e, 0x44, 0x46, 0x6b,
				0x4d, 0x6a, 0x46, 0x69, 0x4d, 0x57, 0x4d, 0x7a,
				0x5a, 0x57, 0x45, 0x33, 0x59, 0x6a, 0x51, 0x35,
				0x4d, 0x6d, 0x4d, 0x78, 0x5a, 0x54, 0x6b, 0x35,
				0x5a, 0x44, 0x4d, 0x7a, 0x5a, 0x47, 0x56, 0x69,
				0x4d, 0x7a, 0x64, 0x6c, 0x4d, 0x32, 0x59, 0x30,
				0x4d, 0x6a, 0x49, 0x78, 0x4d, 0x32, 0x51, 0x77,
				0x59, 0x32, 0x55, 0x32, 0x4e, 0x7a, 0x4d, 0x35,
				0x4e, 0x54, 0x63, 0x30, 0x5a, 0x57, 0x4e, 0x69,
				0x59, 0x7a, 0x59, 0x69, 0x4c, 0x43, 0x4a, 0x32,
				0x5a, 0x58, 0x4a, 0x7a, 0x61, 0x57, 0x39, 0x75,
				0x49, 0x6a, 0x6f, 0x79, 0x4c, 0x43, 0x4a, 0x30,
				0x61, 0x57, 0x31, 0x6c, 0x49, 0x6a, 0x6f, 0x78,
				0x4e, 0x54, 0x55, 0x79, 0x4f, 0x54, 0x51, 0x34,
				0x4e, 0x44, 0x67, 0x31, 0x66, 0x51, 0x3d, 0x3d},
		},
	}
	anchorAggregation := cal.AggregateAnchorTx(aggregationItems)
	if anchorAggregation.AnchorBtcAggID != "d0c045a6-2fee-7310-b58c-58a7e6e3b594" {
		t.Errorf("Aggregation UUIDs don't match; expected %s, got %s\n", "d0c045a6-2fee-7310-b58c-58a7e6e3b594", anchorAggregation.AnchorBtcAggID)
	}
	if anchorAggregation.AnchorBtcAggRoot != "d0c045a62fee7310b58c58a7e6e3b5948ef59c8f86454898149cd578609f515f" {
		t.Errorf("Aggregation Roots don't match; expected %s, got %s\n", "d0c045a62fee7310b58c58a7e6e3b5948ef59c8f86454898149cd578609f515f", anchorAggregation.AnchorBtcAggRoot)

	}
}

func TestQueueCalStateMessage(t *testing.T) {
	assert := assert.New(t)
	time.Sleep(5 * time.Second) //sleep until rabbit comes online
	rabbitTestURI := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	hashBytes, err := hex.DecodeString("B52E3F769921B60352F750BFF7998A1D6E3159494C32C639E0431439D418DBBA")
	assert.Equal(nil, err, "DecodeString of Hash string err should be nil")
	dataBytes, err := base64.StdEncoding.DecodeString("eyJ0eXBlIjoiQ0FMIiwiZGF0YSI6IjBhYzFkNjU2ZWMwYzVkOWQwYWUxNjM1ZTk3MGJiMTMxZDk4Yjk0MTBjMDNhY2U2OWNmNTEwNjk5ZDJhOWNjYjEiLCJ2ZXJzaW9uIjoyLCJ0aW1lIjoxNTUzMTkwODI1fQ==")
	assert.Equal(nil, err, "DecodeString of Data string err should be nil")
	txTm := types.TxTm{
		Hash: hashBytes,
		Data: dataBytes,
	}
	cal := NewCalendar(rabbitTestURI)
	calAggregation := types.CalAgg{CalRoot: "80e157fd9029660d2900391b003ee0017671431e5e37fb46dc252877e13355ec", ProofData: []types.CalProofData{
		types.CalProofData{AggID: "f4c5445a-49ca-11e9-b3cf-0242ac190005", Proof: []types.ProofLineItem{
			types.ProofLineItem{Left: "", Right: "e05d2a67a74278dd4bca5866f4c5a5e66ba7a2217205959bfb27fc0e727aa7d0", Op: ""},
			types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}},
		types.CalProofData{AggID: "f4c549c3-49ca-11e9-b3cf-0242ac190005", Proof: []types.ProofLineItem{
			types.ProofLineItem{Left: "89b4f6c13d489cd5e5bd557234cf00d4daeedc2dd50a1f18e525296e5abd1399", Right: "", Op: ""},
			types.ProofLineItem{Left: "", Right: "", Op: "sha-256"}}}}}

	cal.QueueCalStateMessage(txTm, calAggregation)

	session, err := rabbitmq.ConnectAndConsume(rabbitTestURI, "work.proofstate")
	for m := range session.Msgs {
		assert.Equal(m.Type, "cal_batch", "rabbitmq message type should be cal_batch")
		var stateObj types.CalState
		err = json.Unmarshal(m.Body, &stateObj)
		assert.Equal(err, nil, "err upon unmarshalling calState data should be nil")
		assert.Equal(strings.ToUpper(stateObj.CalID), "B52E3F769921B60352F750BFF7998A1D6E3159494C32C639E0431439D418DBBA", "BTC Tx hash should match between monitor output and rabbit message")
		m.Ack(false)
		break
	}
	err = session.End()
}

func TestQueueBtcaStateDataMessage(t *testing.T) {
	assert := assert.New(t)
	time.Sleep(5 * time.Second) //sleep until rabbit comes online
	rabbitTestURI := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	cal := NewCalendar(rabbitTestURI)
	btcAgg := types.BtcAgg{
		AnchorBtcAggID:   "blah",
		AnchorBtcAggRoot: "hurr",
	}
	err := cal.QueueBtcaStateDataMessage(true, btcAgg)
	assert.Equal(nil, err, "output of QueueBtcaStateDataMessage should be nil")
	// Test that object is in proofstate queue
	session, err := rabbitmq.ConnectAndConsume(rabbitTestURI, "work.proofstate")
	for m := range session.Msgs {
		assert.Equal(m.Type, "anchor_btc_agg_batch", "rabbitmq message type should be agg_batch")
		var stateObj types.BtcAgg
		err = json.Unmarshal(m.Body, &stateObj)
		assert.Equal(err, nil, "err upon unmarshalling btcAgg data should be nil")
		assert.Equal(stateObj.AnchorBtcAggRoot, "hurr", "output btcAgg should match input")
		m.Ack(false)
		break
	}
	err = session.End()
	// Test that object is in btctx queue since we're leader
	session, err = rabbitmq.ConnectAndConsume(rabbitTestURI, "work.btctx")
	for m := range session.Msgs {
		assert.Equal(m.Type, "", "rabbitmq message type should be empty string")
		var stateObj types.BtcAgg
		err = json.Unmarshal(m.Body, &stateObj)
		assert.Equal(err, nil, "err upon unmarshalling btcAgg data should be nil")
		assert.Equal(stateObj.AnchorBtcAggRoot, "hurr", "output btcAgg should match input")
		m.Ack(false)
		break
	}
	err = session.End()
}
