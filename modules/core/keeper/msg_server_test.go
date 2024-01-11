package keeper_test

import (
	"errors"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

var (
	timeoutHeight = clienttypes.NewHeight(1, 10000)
	maxSequence   = uint64(10)
)

// tests the IBC handler receiving a packet on ordered and unordered channels.
// It verifies that the storing of an acknowledgement on success occurs. It
// tests high level properties like ordering and basic sanity checks. More
// rigorous testing of 'RecvPacket' can be found in the
// 04-channel/keeper/packet_test.go.
func (suite *KeeperTestSuite) TestHandleRecvPacket() {
	var (
		packet channeltypes.Packet
		path   *ibctesting.Path
	)

	testCases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expRevert bool
		async     bool // indicate no ack written
		replay    bool // indicate replay (no-op)
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false, false, false},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false, false, false},
		{"success: UNORDERED out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, true, false, false, false},
		{"success: OnRecvPacket callback returns revert=true", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockFailPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockFailPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, true, false, false},
		{"success: ORDERED - async acknowledgement", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false, true, false},
		{"success: UNORDERED - async acknowledgement", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false, true, false},
		{"failure: ORDERED out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, false, false, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, false, false, false, false},
		{"packet not sent", func() {
			suite.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, false, false, false, false},
		{"successful no-op: ORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true, false, false, true},
		{"successful no-op: UNORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true, false, false, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			if path.EndpointA.ChannelID != "" {
				proof, proofHeight = path.EndpointA.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())

			ctx := suite.chainB.GetContext()
			_, err := keeper.Keeper.RecvPacket(*suite.chainB.App.GetIBCKeeper(), ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expPass {
				suite.Require().NoError(err)

				// replay should not fail since it will be treated as a no-op
				_, err := keeper.Keeper.RecvPacket(*suite.chainB.App.GetIBCKeeper(), suite.chainB.GetContext(), msg)
				suite.Require().NoError(err)

				// check that callback state was handled correctly
				_, exists := suite.chainB.GetSimApp().ScopedIBCMockKeeper.GetCapability(suite.chainB.GetContext(), ibcmock.GetMockRecvCanaryCapabilityName(packet))
				if tc.expRevert {
					suite.Require().False(exists, "capability exists in store even after callback reverted")

					// context events should contain error events
					suite.Require().Contains(events, keeper.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					suite.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
				} else {
					suite.Require().True(exists, "callback state not persisted when revert is false")

					if tc.replay {
						// context should not contain application events
						suite.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
						suite.Require().NotContains(events, keeper.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					} else {
						// context events should contain application events
						suite.Require().Contains(events, ibcmock.NewMockRecvPacketEvent())
					}
				}

				// verify if ack was written
				ack, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if tc.async {
					suite.Require().Nil(ack)
					suite.Require().False(found)

				} else {
					suite.Require().NotNil(ack)
					suite.Require().True(found)
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// tests the IBC handler acknowledgement of a packet on ordered and unordered
// channels. It verifies that the deletion of packet commitments from state
// occurs. It test high level properties like ordering and basic sanity
// checks. More rigorous testing of 'AcknowledgePacket'
// can be found in the 04-channel/keeper/packet_test.go.
func (suite *KeeperTestSuite) TestHandleAcknowledgePacket() {
	var (
		packet channeltypes.Packet
		path   *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
		replay   bool // indicate replay (no-op)
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true, false},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true, false},
		{"success: UNORDERED acknowledge out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment)
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, true, false},
		{"failure: ORDERED acknowledge out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, false, false},
		{"packet not received", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, false, false},
		{"successful no-op: ORDERED - packet already acknowledged (replay)", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, true, true},
		{"successful no-op: UNORDERED - packet already acknowledged (replay)", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, true, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgAcknowledgement(packet, ibcmock.MockAcknowledgement.Acknowledgement(), proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			ctx := suite.chainA.GetContext()
			_, err := keeper.Keeper.Acknowledgement(*suite.chainA.App.GetIBCKeeper(), ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expPass {
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				// replay should not error as it is treated as a no-op
				_, err := keeper.Keeper.Acknowledgement(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				if tc.replay {
					// context should not contain application events
					suite.Require().NotContains(events, ibcmock.NewMockAckPacketEvent())
				} else {
					// context events should contain application events
					suite.Require().Contains(events, ibcmock.NewMockAckPacketEvent())
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// tests the IBC handler timing out a packet on ordered and unordered channels.
// It verifies that the deletion of a packet commitment occurs. It tests
// high level properties like ordering and basic sanity checks. More
// rigorous testing of 'TimeoutPacket' and 'TimeoutExecuted' can be found in
// the 04-channel/keeper/timeout_test.go.
func (suite *KeeperTestSuite) TestHandleTimeoutPacket() {
	var (
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
		noop     bool // indicate no-op
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, true, false},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, true, false},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, true, false},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, true, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false, false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			suite.coordinator.Setup(path)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, true, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgTimeout(packet, 1, proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			ctx := suite.chainA.GetContext()
			_, err := keeper.Keeper.Timeout(*suite.chainA.App.GetIBCKeeper(), ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expPass {
				suite.Require().NoError(err)

				// replay should not return an error as it is treated as a no-op
				_, err := keeper.Keeper.Timeout(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				if tc.noop {
					// context should not contain application events
					suite.Require().NotContains(events, ibcmock.NewMockTimeoutPacketEvent())
				} else {
					// context should contain application events
					suite.Require().Contains(events, ibcmock.NewMockTimeoutPacketEvent())
				}

			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// tests the IBC handler timing out a packet via channel closure on ordered
// and unordered channels. It verifies that the deletion of a packet
// commitment occurs. It tests high level properties like ordering and basic
// sanity checks. More rigorous testing of 'TimeoutOnClose' and
// 'TimeoutExecuted' can be found in the 04-channel/keeper/timeout_test.go.
func (suite *KeeperTestSuite) TestHandleTimeoutOnClosePacket() {
	var (
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			suite.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err := path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"ORDERED: channel not closed", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			proof, proofHeight := suite.chainB.QueryProof(packetKey)

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			proofClosed, _ := suite.chainB.QueryProof(channelKey)

			msg := channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, proofClosed, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			_, err := keeper.Keeper.TimeoutOnClose(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			if tc.expPass {
				suite.Require().NoError(err)

				// replay should not return an error as it will be treated as a no-op
				_, err := keeper.Keeper.TimeoutOnClose(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpgradeClient() {
	var (
		path              *ibctesting.Path
		newChainID        string
		newClientHeight   clienttypes.Height
		upgradedClient    exported.ClientState
		upgradedConsState exported.ConsensusState
		lastHeight        exported.Height
		msg               *clienttypes.MsgUpgradeClient
	)
	cases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			name: "successful upgrade",
			setup: func() {
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+ibctesting.TrustingPeriod, ibctesting.MaxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
				upgradedClient = upgradedClient.ZeroCustomFields()

				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("nextValsHash"),
				}

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
				suite.Require().NoError(err)

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradeClient, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState,
					proofUpgradeClient, proofUpgradedConsState, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "VerifyUpgrade fails",
			setup: func() {
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+ibctesting.TrustingPeriod, ibctesting.MaxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
				upgradedClient = upgradedClient.ZeroCustomFields()

				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("nextValsHash"),
				}

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
				suite.Require().NoError(err)

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState, nil, nil, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			expPass: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		suite.coordinator.SetupClients(path)

		var err error
		clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		suite.Require().NoError(err)

		newClientHeight = clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1)

		tc.setup()

		_, err = keeper.Keeper.UpgradeClient(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

		if tc.expPass {
			suite.Require().NoError(err, "upgrade handler failed on valid case: %s", tc.name)
			newClient, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
			suite.Require().True(ok)
			newChainSpecifiedClient := newClient.ZeroCustomFields()
			suite.Require().Equal(upgradedClient, newChainSpecifiedClient)
		} else {
			suite.Require().Error(err, "upgrade handler passed on invalid case: %s", tc.name)
		}
	}
}

func (suite *KeeperTestSuite) TestRecoverClient() {
	var msg *clienttypes.MsgRecoverClient

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: recover client",
			func() {},
			nil,
		},
		{
			"signer doesn't match authority",
			func() {
				msg.Signer = ibctesting.InvalidID
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"invalid subject client",
			func() {
				msg.SubjectClientId = ibctesting.InvalidID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(subjectPath)
			subject := subjectPath.EndpointA.ClientID
			subjectClientState := suite.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(substitutePath)
			substitute := substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

			msg = clienttypes.NewMsgRecoverClient(suite.chainA.App.GetIBCKeeper().GetAuthority(), subject, substitute)

			tc.malleate()

			_, err = keeper.Keeper.RecoverClient(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// Assert that client status is now Active
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				tmClientState := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().Equal(tmClientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec()), exported.Active)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestIBCSoftwareUpgrade tests the IBCSoftwareUpgrade rpc handler
func (suite *KeeperTestSuite) TestIBCSoftwareUpgrade() {
	var msg *clienttypes.MsgIBCSoftwareUpgrade
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid authority and client upgrade",
			func() {},
			nil,
		},
		{
			"failure: invalid authority address",
			func() {
				msg.Signer = suite.chainA.SenderAccount.GetAddress().String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: invalid clientState",
			func() {
				msg.UpgradedClientState = nil
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"failure: failed to schedule client upgrade",
			func() {
				msg.Plan.Height = 0
			},
			sdkerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(path)
			validAuthority := suite.chainA.App.GetIBCKeeper().GetAuthority()
			plan := upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}
			// update trusting period
			clientState := path.EndpointB.GetClientState()
			clientState.(*ibctm.ClientState).TrustingPeriod += 100

			var err error
			msg, err = clienttypes.NewMsgIBCSoftwareUpgrade(
				validAuthority,
				plan,
				clientState,
			)

			suite.Require().NoError(err)

			tc.malleate()

			_, err = keeper.Keeper.IBCSoftwareUpgrade(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			if tc.expError == nil {
				suite.Require().NoError(err)
				// upgrade plan is stored
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				suite.Require().NoError(err)
				suite.Require().Equal(plan, storedPlan)

				// upgraded client state is stored
				bz, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().NoError(err)
				upgradedClientState, err := clienttypes.UnmarshalClientState(suite.chainA.App.AppCodec(), bz)
				suite.Require().NoError(err)
				suite.Require().Equal(clientState.ZeroCustomFields(), upgradedClientState)
			} else {
				suite.Require().True(errors.Is(err, tc.expError))
			}
		})
	}
}

// TestUpdateClientParams tests the UpdateClientParams rpc handler
func (suite *KeeperTestSuite) TestUpdateClientParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *clienttypes.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid signer and default params",
			clienttypes.NewMsgUpdateParams(signer, clienttypes.DefaultParams()),
			true,
		},
		{
			"failure: malformed signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.InvalidID, clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: empty signer address",
			clienttypes.NewMsgUpdateParams("", clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: whitespace signer address",
			clienttypes.NewMsgUpdateParams("    ", clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.TestAccAddress, clienttypes.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := keeper.Keeper.UpdateClientParams(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), tc.msg)
			if tc.expPass {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestUpdateConnectionParams tests the UpdateConnectionParams rpc handler
func (suite *KeeperTestSuite) TestUpdateConnectionParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *connectiontypes.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid signer and default params",
			connectiontypes.NewMsgUpdateParams(signer, connectiontypes.DefaultParams()),
			true,
		},
		{
			"failure: malformed signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.InvalidID, connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: empty signer address",
			connectiontypes.NewMsgUpdateParams("", connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: whitespace signer address",
			connectiontypes.NewMsgUpdateParams("    ", connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.TestAccAddress, connectiontypes.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := keeper.Keeper.UpdateConnectionParams(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), tc.msg)
			if tc.expPass {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
