package wasm_test

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	_go "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
	wasm "github.com/cosmos/ibc-go/v5/modules/light-clients/10-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	"github.com/cosmos/ibc-go/v5/testing/simapp"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type WasmTestSuite struct {
	suite.Suite
	coordinator *ibctesting.Coordinator
	wasm        *ibctesting.Wasm // singlesig public key
	// Tendermint chain
	chainA *ibctesting.TestChain
	// Grandpa chain
	chainB         *ibctesting.TestChain
	ctx            sdk.Context
	cdc            codec.Codec
	now            time.Time
	store          sdk.KVStore
	clientState    wasm.ClientState
	consensusState wasm.ConsensusState
	codeId         []byte
	testData       map[string]string
}

func (suite *WasmTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	suite.wasm = ibctesting.NewWasm(suite.T(), suite.chainA.Codec, "wasmsingle", "testing", 1)
	// suite.solomachineMulti = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinemulti", "testing", 4)

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(checkTx)
	suite.cdc = app.AppCodec()
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	data, err := os.ReadFile("test_data/raw.json")
	suite.Require().NoError(err)
	err = json.Unmarshal(data, &suite.testData)
	suite.Require().NoError(err)

	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1, Time: suite.now}).WithGasMeter(sdk.NewInfiniteGasMeter())
	wasmConfig := wasm.VMConfig{
		DataDir:           "tmp",
		SupportedFeatures: []string{"storage", "iterator"},
		MemoryLimitMb:     uint32(math.Pow(2, 12)),
		PrintDebug:        true,
		CacheSizeMb:       uint32(math.Pow(2, 8)),
	}
	validationConfig := wasm.ValidationConfig{
		MaxSizeAllowed: int(math.Pow(2, 26)),
	}
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), exported.Wasm)
	data, err = hex.DecodeString(suite.testData["client_state_a0"])
	suite.Require().NoError(err)
	clientState := wasm.ClientState{
		Data: data,
		LatestHeight: &clienttypes.Height{
			RevisionNumber: 1,
			RevisionHeight: 2,
		},
		ProofSpecs: []*_go.ProofSpec{
			{
				LeafSpec: &_go.LeafOp{
					Hash:         _go.HashOp_SHA256,
					Length:       _go.LengthOp_FIXED32_BIG,
					PrehashValue: _go.HashOp_SHA256,
					Prefix:       []byte{0},
				},
				InnerSpec: &_go.InnerSpec{
					ChildOrder:      []int32{0, 1},
					ChildSize:       33,
					MinPrefixLength: 4,
					MaxPrefixLength: 12,
					EmptyChild:      nil,
					Hash:            _go.HashOp_SHA256,
				},
				MaxDepth: 0,
				MinDepth: 0,
			},
		},
		Repository: "test",
	}
	os.MkdirAll("tmp", 0o755)
	wasm.CreateVM(&wasmConfig, &validationConfig)
	data, err = os.ReadFile("ics10_grandpa_cw.wasm")
	suite.Require().NoError(err)

	err = wasm.PushNewWasmCode(suite.store, &clientState, data)
	suite.Require().NoError(err)
	suite.clientState = clientState
	data, err = hex.DecodeString(suite.testData["consensus_state_a0"])
	suite.Require().NoError(err)
	consensusState := wasm.ConsensusState{
		Data:      data,
		CodeId:    clientState.CodeId,
		Timestamp: uint64(suite.now.UnixNano()),
		Root: &commitmenttypes.MerkleRoot{
			Hash: []byte{0},
		},
	}
	suite.consensusState = consensusState
	suite.codeId = clientState.CodeId
	// err = clientState.Initialize(suite.ctx, suite.cdc, suite.store, &consensusState)
	// suite.Require().NoError(err)

	// err = clientState.VerifyClientMessage()
	/*
		path := ibctesting.NewPath(suite.chainA, suite.chainB)
		// path.EndpointA.ClientID = "unnamed_client_a"
		// path.EndpointB.ClientID = "unnamed_client_b"
		// endpointA := ibctesting.NewDefaultEndpoint(suite.chainA)
		// endpointA.ClientID = "unnamed_client_a"
		// endpointB := ibctesting.NewDefaultEndpoint(suite.chainB)
		// endpointB.ClientID = "unnamed_client_b"
		fmt.Println("A", path.EndpointA.ClientConfig.GetClientType())
		path.EndpointB.ClientConfig = ibctesting.NewWasmConfig()
		fmt.Println("B", path.EndpointB.ClientConfig.GetClientType())
		suite.Require().NoError(err)
		msg, err := clienttypes.NewMsgCreateClient(&clientState, &consensusState, path.EndpointA.Chain.SenderAccount.GetAddress().String())
		suite.Require().NoError(err)
		res, err := suite.chainA.SendMsgs(msg)
		suite.Require().NoError(err)
		path.EndpointA.ClientID, err = ibctesting.ParseClientIDFromEvents(res.GetEvents())
		suite.Require().NoError(err)

		suite.Require().NoError(err)
		msg, err = clienttypes.NewMsgCreateClient(&clientState, &consensusState, path.EndpointB.Chain.SenderAccount.GetAddress().String())
		suite.Require().NoError(err)
		res, err = suite.chainB.SendMsgs(msg)
		suite.Require().NoError(err)
		path.EndpointB.ClientID, err = ibctesting.ParseClientIDFromEvents(res.GetEvents())
		suite.Require().NoError(err)

		err = path.EndpointA.ConnOpenInit()
		suite.Require().NoError(err)

		err = path.EndpointB.ConnOpenTry()
		suite.Require().NoError(err)

		err = path.EndpointA.ConnOpenAck()
		suite.Require().NoError(err)

		err = path.EndpointB.ConnOpenConfirm()
		suite.Require().NoError(err)

		// ensure counterparty is up to date
		// err = path.EndpointA.UpdateClient()
		// suite.Require().NoError(err)

		// header := wasm.Header{
		// 	Data: []byte{0},
		// 	Height: &clienttypes.Height{
		// 		RevisionNumber: 1,
		// 		RevisionHeight: 2,
		// 	},
		// }
		// msg, err := clienttypes.NewMsgUpdateClient(
		// 	endpointA.ClientID, &header,
		// 	suite.chainA.SenderAccount.GetAddress().String(),
		// )
		// endpointA.ClientConfig = &ibctesting.WasmConfig{
		// 	InitClientState:    clientState,
		// 	InitConsensusState: consensusState,
		// }
		println(res)
	*/
}

func (suite *WasmTestSuite) TestVerifyClientMessageHeader() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasm.ClientState
	)

	// test singlesig and multisig public keys
	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful header",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasm.Header{
						Data: data,
						Height: &clienttypes.Height{
							RevisionNumber: 1,
							RevisionHeight: 2,
						},
					}
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				tc.setup()

				clientState = &suite.clientState
				err := clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}

func (suite *WasmTestSuite) TestUpdateState() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasm.ClientState
	)

	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful update",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasm.Header{
						Data: data,
						Height: &clienttypes.Height{
							RevisionNumber: 1,
							RevisionHeight: 2,
						},
					}
					clientState = &suite.clientState
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()

				if tc.expPass {
					consensusHeights := clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

					clientStateBz := suite.store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

					suite.Require().Len(consensusHeights, 1)
					suite.Require().Equal(&clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 89,
					}, consensusHeights[0])
					suite.Require().Equal(consensusHeights[0], newClientState.(*wasm.ClientState).LatestHeight)
				} else {
					suite.Require().Panics(func() {
						clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
					})
				}
			})
		}
	}
}

func (suite *WasmTestSuite) TestVerifyMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasm.ClientState
	)

	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful update",
				func() {
					data, err := hex.DecodeString(suite.testData["misbehaviour_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasm.Misbehaviour{
						ClientId: wm.ClientID,
						Data:     data,
					}
					clientState = &suite.clientState
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()
				println(clientMsg, clientState)
				// TODO: uncomment when fisherman is merged
				/*
					err := clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

					if tc.expPass {
						suite.Require().NoError(err)
					} else {
						suite.Require().Error(err)
					}
				*/
			})
		}
	}
}

func (suite *WasmTestSuite) TestWasm() {
	suite.Run("Init contract", func() {
		suite.SetupTest()
	})
}

func TestWasmTestSuite(t *testing.T) {
	suite.Run(t, new(WasmTestSuite))
}
