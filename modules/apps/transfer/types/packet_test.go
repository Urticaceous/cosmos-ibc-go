package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

const (
	denom              = "transfer/gaiachannel/atom"
	amount             = "100"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

// TestFungibleTokenPacketDataValidateBasic tests ValidateBasic for FungibleTokenPacketData
func TestFungibleTokenPacketDataValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expPass    bool
	}{
		{"valid packet", types.NewFungibleTokenPacketData(denom, amount, addr1, addr2, ""), true},
		{"valid packet with memo", types.NewFungibleTokenPacketData(denom, amount, addr1, addr2, "memo"), true},
		{"valid packet with large amount", types.NewFungibleTokenPacketData(denom, largeAmount, addr1, addr2, ""), true},
		{"invalid denom", types.NewFungibleTokenPacketData("", amount, addr1, addr2, ""), false},
		{"invalid empty amount", types.NewFungibleTokenPacketData(denom, "", addr1, addr2, ""), false},
		{"invalid zero amount", types.NewFungibleTokenPacketData(denom, "0", addr1, addr2, ""), false},
		{"invalid negative amount", types.NewFungibleTokenPacketData(denom, "-1", addr1, addr2, ""), false},
		{"invalid large amount", types.NewFungibleTokenPacketData(denom, invalidLargeAmount, addr1, addr2, ""), false},
		{"missing sender address", types.NewFungibleTokenPacketData(denom, amount, emptyAddr, addr2, ""), false},
		{"missing recipient address", types.NewFungibleTokenPacketData(denom, amount, addr1, emptyAddr, ""), false},
	}

	for i, tc := range testCases {
		err := tc.packetData.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
<<<<<<< HEAD
=======

func (suite *TypesTestSuite) TestGetPacketSender() {
	packetData := types.FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     "",
	}

	suite.Require().Equal(sender, packetData.GetPacketSender(types.PortID))
}

func (suite *TypesTestSuite) TestPacketDataProvider() {
	testCases := []struct {
		name          string
		packetData    types.FungibleTokenPacketData
		expCustomData interface{}
	}{
		{
			"success: src_callback key in memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver),
			},
			map[string]interface{}{
				"address": receiver,
			},
		},
		{
			"success: src_callback key in memo with additional fields",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, receiver),
			},
			map[string]interface{}{
				"address":   receiver,
				"gas_limit": "200000",
			},
		},
		{
			"success: src_callback has string value",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": "string"}`,
			},
			"string",
		},
		{
			"failure: empty memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			nil,
		},
		{
			"failure: non-json memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "invalid",
			},
			nil,
		},
	}

	for _, tc := range testCases {
		customData := tc.packetData.GetCustomPacketData("src_callback")
		suite.Require().Equal(tc.expCustomData, customData)
	}
}
>>>>>>> ec684384 (feat(core, apps): 'PacketData' interface added and implemented (#4200))
