package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// InitInterchainAccount is the entry point to registering an interchain account.
// It generates a new port identifier using the owner address, connection identifier,
// and counterparty connection identifier. It will bind to the port identifier and
// call 04-channel 'ChanOpenInit'. An error is returned if the port identifier is
// already in use. Gaining access to interchain accounts whose channels have closed
// cannot be done with this function. A regular MsgChanOpenInit must be used.
func (k Keeper) InitInterchainAccount(ctx sdk.Context, connectionID, counterpartyConnectionID, owner string) error {
	portId, err := types.GeneratePortID(owner, connectionID, counterpartyConnectionID)
	if err != nil {
		return err
	}

	// check if the port is already bound
	if k.IsBound(ctx, portId) {
		return sdkerrors.Wrap(types.ErrPortAlreadyBound, portId)
	}

	portCap := k.portKeeper.BindPort(ctx, portId)
	err = k.ClaimCapability(ctx, portCap, host.PortPath(portId))
	if err != nil {
		return sdkerrors.Wrap(err, "unable to bind to newly generated portID")
	}

	msg := channeltypes.NewMsgChannelOpenInit(portId, types.Version, channeltypes.ORDERED, []string{connectionID}, types.PortID, types.ModuleName)
	handler := k.msgRouter.Handler(msg)
	if _, err := handler(ctx, msg); err != nil {
		return err
	}

	return nil
}

// Register interchain account if it has not already been created
func (k Keeper) RegisterInterchainAccount(ctx sdk.Context, portId string) {
	address := types.GenerateAddress(portId)

	account := k.accountKeeper.GetAccount(ctx, address)
	if account != nil {
		// account already created, return no-op
		return
	}

	interchainAccount := types.NewInterchainAccount(
		authtypes.NewBaseAccountWithAddress(address),
		portId,
	)

	k.accountKeeper.NewAccount(ctx, interchainAccount)
	k.accountKeeper.SetAccount(ctx, interchainAccount)
	_ = k.SetInterchainAccountAddress(ctx, portId, interchainAccount.Address)
}

func (k Keeper) GetInterchainAccount(ctx sdk.Context, addr sdk.AccAddress) (types.InterchainAccount, error) {
	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return types.InterchainAccount{}, sdkerrors.Wrap(types.ErrInterchainAccountNotFound, "there is no account")
	}

	interchainAccount, ok := acc.(*types.InterchainAccount)
	if !ok {
		return types.InterchainAccount{}, sdkerrors.Wrap(types.ErrInterchainAccountNotFound, "account is not an interchain account")
	}
	return *interchainAccount, nil
}
