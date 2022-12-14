package sealing_test

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/actors/policy"

	"github.com/filecoin-project/venus-sealer/constants"
	sealing "github.com/filecoin-project/venus-sealer/storage-sealing"
	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type fakeChain struct {
	h abi.ChainEpoch
}

type fakeConfigStub struct {
	CCSectorLifetime time.Duration
}

func fakeConfigGetter(stub *fakeConfigStub) types.GetSealingConfigFunc {
	return func() (sealiface.Config, error) {
		if stub == nil {
			return sealiface.Config{}, nil
		}

		return sealiface.Config{
			CommittedCapacitySectorLifetime: stub.CCSectorLifetime,
		}, nil
	}
}

func (f *fakeChain) StateNetworkVersion(ctx context.Context, tok types.TipSetToken) (network.Version, error) {
	return constants.NewestNetworkVersion, nil
}

func (f *fakeChain) ChainHead(ctx context.Context) (types.TipSetToken, abi.ChainEpoch, error) {
	return []byte{1, 2, 3}, f.h, nil
}

func fakePieceCid(t *testing.T) cid.Cid {
	comm := [32]byte{1, 2, 3}
	fakePieceCid, err := commcid.ReplicaCommitmentV1ToCID(comm[:])
	require.NoError(t, err)
	return fakePieceCid
}

func TestBasicPolicyEmptySector(t *testing.T) {
	cfg := fakeConfigGetter(nil)
	h := abi.ChainEpoch(55)
	pBuffer := abi.ChainEpoch(2)
	pcp := sealing.NewBasicPreCommitPolicy(&fakeChain{h: h}, cfg, pBuffer)
	exp, err := pcp.Expiration(context.Background())

	require.NoError(t, err)

	// as set when there are no deal pieces
	expected := h + policy.GetMaxSectorExpirationExtension() - pBuffer
	assert.Equal(t, int(expected), int(exp))
}

func TestCustomCCSectorConfig(t *testing.T) {
	customLifetime := 200 * 24 * time.Hour
	customLifetimeEpochs := abi.ChainEpoch(int64(customLifetime.Seconds()) / builtin.EpochDurationSeconds)
	cfgStub := fakeConfigStub{CCSectorLifetime: customLifetime}
	cfg := fakeConfigGetter(&cfgStub)
	h := abi.ChainEpoch(55)
	pBuffer := abi.ChainEpoch(2)
	pcp := sealing.NewBasicPreCommitPolicy(&fakeChain{h: h}, cfg, pBuffer)
	exp, err := pcp.Expiration(context.Background())

	require.NoError(t, err)

	// as set when there are no deal pieces
	expected := h + customLifetimeEpochs - pBuffer
	assert.Equal(t, int(expected), int(exp))
}

func TestBasicPolicyMostConstrictiveSchedule(t *testing.T) {
	cfg := fakeConfigGetter(nil)
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, cfg, 2)
	longestDealEpochEnd := abi.ChainEpoch(547300)
	pieces := []types.Piece{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.PieceDealInfo{
				DealID: abi.DealID(42),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(70),
					EndEpoch:   abi.ChainEpoch(547275),
				},
			},
		},
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.PieceDealInfo{
				DealID: abi.DealID(43),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(80),
					EndEpoch:   longestDealEpochEnd,
				},
			},
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 2890, int(exp))
}

func TestBasicPolicyIgnoresExistingScheduleIfExpired(t *testing.T) {
	cfg := fakeConfigGetter(nil)
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, cfg, 0)

	pieces := []types.Piece{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.PieceDealInfo{
				DealID: abi.DealID(44),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(1),
					EndEpoch:   abi.ChainEpoch(10),
				},
			},
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	// Treated as a CC sector, so expiration becomes currEpoch + maxLifetime = 55 + 1555200
	assert.Equal(t, 1555255, int(exp))
}

func TestMissingDealIsIgnored(t *testing.T) {
	cfg := fakeConfigGetter(nil)
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, cfg, 0)

	pieces := []types.Piece{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.PieceDealInfo{
				DealID: abi.DealID(44),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(1),
					EndEpoch:   abi.ChainEpoch(547300),
				},
			},
		},
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: nil,
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 547300, int(exp))
}
