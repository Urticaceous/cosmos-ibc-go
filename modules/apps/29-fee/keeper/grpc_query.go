package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

var _ types.QueryServer = Keeper{}

// IncentivizedPackets implements the IncentivizedPackets gRPC method
func (k Keeper) IncentivizedPackets(c context.Context, req *types.QueryIncentivizedPacketsRequest) (*types.QueryIncentivizedPacketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c).WithBlockHeight(int64(req.QueryHeight))

	packets := []*types.IdentifiedPacketFee{}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.FeeInEscrowPrefix))
	_, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		result := k.MustUnmarshalFee(value)
		packets = append(packets, &result)
		return nil
	})

	if err != nil {
		return nil, status.Error(
			codes.NotFound, err.Error(),
		)
	}

	return &types.QueryIncentivizedPacketsResponse{
		IncentivizedPackets: packets,
	}, nil
}

// IncentivizedPacket implements the IncentivizedPacket gRPC method
func (k Keeper) IncentivizedPacket(c context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c).WithBlockHeight(int64(req.QueryHeight))

	fee, exists := k.GetFeeInEscrow(ctx, req.PacketId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrFeeNotFound, req.PacketId.String()).Error(),
		)
	}

	return &types.QueryIncentivizedPacketResponse{
		IncentivizedPacket: &fee,
	}, nil
}
