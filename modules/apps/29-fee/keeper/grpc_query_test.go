package keeper_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPackets() {
	var (
		req             *types.QueryIncentivizedPacketsRequest
		expectedPackets []types.IdentifiedPacketFees
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

				fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
				packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

				for i := 0; i < 3; i++ {
					// escrow packet fees for three different packets
					packetID := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, uint64(i+1))
					suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)

					expectedPackets = append(expectedPackets, types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee}))
				}

				req = &types.QueryIncentivizedPacketsRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"empty pagination",
			func() {
				expectedPackets = nil
				req = &types.QueryIncentivizedPacketsRequest{}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate() // malleate mutates test data

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.IncentivizedPackets(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expectedPackets, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {
	var (
		req *types.QueryIncentivizedPacketRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"fees not found for packet id",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 100),
					QueryHeight: 0,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1)
			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			for i := 0; i < 3; i++ {
				// escrow three packet fees for the same packet
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
				suite.Require().NoError(err)
			}

			req = &types.QueryIncentivizedPacketRequest{
				PacketId:    packetID,
				QueryHeight: 0,
			}

			tc.malleate() // malleate mutates test data

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.IncentivizedPacket(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee, packetFee, packetFee}), res.IncentivizedPacket)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacketsForChannel() {
	var (
		req                     *types.QueryIncentivizedPacketsForChannelRequest
		expIdentifiedPacketFees []*types.IdentifiedPacketFees
	)

	fee := types.Fee{
		AckFee:     sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		RecvFee:    sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		TimeoutFee: sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
	}

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      ibctesting.MockFeePort,
					ChannelId:   ibctesting.FirstChannelID,
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"no packets for specified channel",
			func() {
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      ibctesting.MockFeePort,
					ChannelId:   "channel-10",
					QueryHeight: 0,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			// setup
			refundAcc := suite.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), nil)
			packetFees := types.NewPacketFees([]types.PacketFee{packetFee, packetFee, packetFee})

			identifiedFees1 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1), packetFees.PacketFees)
			identifiedFees2 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 2), packetFees.PacketFees)
			identifiedFees3 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 3), packetFees.PacketFees)

			expIdentifiedPacketFees = append(expIdentifiedPacketFees, &identifiedFees1, &identifiedFees2, &identifiedFees3)

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
			for _, identifiedPacketFees := range expIdentifiedPacketFees {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), identifiedPacketFees.PacketId, types.NewPacketFees(identifiedPacketFees.PacketFees))
			}

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())

			res, err := suite.queryClient.IncentivizedPacketsForChannel(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expIdentifiedPacketFees, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalRecvFees() {
	var (
		req *types.QueryTotalRecvFeesRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1)

			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			for i := 0; i < 3; i++ {
				// escrow three packet fees for the same packet
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
				suite.Require().NoError(err)
			}

			req = &types.QueryTotalRecvFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.TotalRecvFees(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default recv fee
				expectedFees := defaultReceiveFee.Add(defaultReceiveFee...).Add(defaultReceiveFee...)
				suite.Require().Equal(expectedFees, res.RecvFees)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalAckFees() {
	var (
		req *types.QueryTotalAckFeesRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1)

			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			for i := 0; i < 3; i++ {
				// escrow three packet fees for the same packet
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
				suite.Require().NoError(err)
			}

			req = &types.QueryTotalAckFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.TotalAckFees(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultAckFee.Add(defaultAckFee...).Add(defaultAckFee...)
				suite.Require().Equal(expectedFees, res.AckFees)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalTimeoutFees() {
	var (
		req *types.QueryTotalTimeoutFeesRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1)

			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			for i := 0; i < 3; i++ {
				// escrow three packet fees for the same packet
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
				suite.Require().NoError(err)
			}

			req = &types.QueryTotalTimeoutFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.TotalTimeoutFees(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultTimeoutFee.Add(defaultTimeoutFee...).Add(defaultTimeoutFee...)
				suite.Require().Equal(expectedFees, res.TimeoutFees)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCounterpartyAddress() {
	var (
		req *types.QueryCounterpartyAddressRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"not found, invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id"
			},
			false,
		},
		{
			"not found, invalid address",
			func() {
				req.RelayerAddress = "invalid-addr"
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expectedCounterpartyAddr := sdk.AccAddress(pk.Address())

			suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(
				suite.chainA.GetContext(),
				suite.chainA.SenderAccount.GetAddress().String(),
				expectedCounterpartyAddr.String(),
				suite.path.EndpointA.ChannelID,
			)

			req = &types.QueryCounterpartyAddressRequest{
				ChannelId:      suite.path.EndpointA.ChannelID,
				RelayerAddress: suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.CounterpartyAddress(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expectedCounterpartyAddr.String(), res.CounterpartyAddress)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
