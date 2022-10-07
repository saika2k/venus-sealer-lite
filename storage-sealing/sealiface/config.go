package sealiface

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"
)

// this has to be in a separate package to not make lotus API depend on filecoin-ffi

type Config struct {
	// 0 = no limit
	MaxWaitDealsSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectorsForDeals uint64

	PreferNewSectorsForDeals bool

	MaxUpgradingSectors uint64

	MakeNewSectorForDeals bool

	MakeCCSectorsAvailable bool

	WaitDealsDelay time.Duration

	CommittedCapacitySectorLifetime time.Duration

	StartEpochSealingBuffer abi.ChainEpoch

	AlwaysKeepUnsealedCopy bool

	FinalizeEarly bool

	CollateralFromMinerBalance bool
	AvailableBalanceBuffer     abi.TokenAmount
	DisableCollateralFallback  bool

	BatchPreCommits     bool
	MaxPreCommitBatch   int
	PreCommitBatchWait  time.Duration
	PreCommitBatchSlack time.Duration

	AggregateCommits bool
	MinCommitBatch   int
	MaxCommitBatch   int
	CommitBatchWait  time.Duration
	CommitBatchSlack time.Duration

	// network BaseFee below which to stop doing commit aggregation, instead
	// submitting proofs to the chain individually
	AggregateAboveBaseFee abi.TokenAmount

	// network BaseFee below which to stop doing precommit batching, instead
	// sending precommit messages to the chain individually
	BatchPreCommitAboveBaseFee abi.TokenAmount

	TerminateBatchMax  uint64
	TerminateBatchMin  uint64
	TerminateBatchWait time.Duration
}