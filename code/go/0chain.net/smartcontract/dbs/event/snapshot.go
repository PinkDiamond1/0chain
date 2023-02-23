package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"

	"go.uber.org/zap"
)

//max_capacity - maybe change it max capacity in blobber config and everywhere else to be less confusing.
//staked - staked capacity by delegates
//unstaked - opportunity for delegates to stake until max capacity
//allocated - clients have locked up storage by purchasing allocations
//unallocated - this is equal to (staked - allocated) and allows clients to purchase allocations with free space blobbers.
//used - this is the actual usage or data that is in the server.
//staked + unstaked = max_capacity
//allocated + unallocated = staked

type Snapshot struct {
	Round int64 `gorm:"primaryKey;autoIncrement:false" json:"round"`

	TotalMint            int64 `json:"total_mint"`
	TotalChallengePools  int64 `json:"total_challenge_pools"`  //486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	ActiveAllocatedDelta int64 `json:"active_allocated_delta"` //496 SUM total amount of new allocation storage in a period (number of allocations active)
	ZCNSupply            int64 `json:"zcn_supply"`             //488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	TotalValueLocked     int64 `json:"total_value_locked"`     //487 SUM Total value locked = Total staked ZCN * Price per ZCN (across all pools)
	ClientLocks          int64 `json:"client_locks"`           //487 SUM How many clients locked in (write/read + challenge)  pools
	MinedTotal           int64 `json:"mined_total"`            // SUM total mined for all providers, never decrease
	// updated from blobber snapshot aggregate table
	AverageWritePrice    int64 `json:"average_write_price"`              //*494 AVG it's the price from the terms and triggered with their updates //???
	TotalStaked          int64 `json:"total_staked"`                     //*485 SUM All providers all pools
	TotalRewards         int64 `json:"total_rewards"`                    //SUM total of all rewards
	SuccessfulChallenges int64 `json:"successful_challenges"`            //*493 SUM percentage of challenges failed by a particular blobber
	TotalChallenges      int64 `json:"total_challenges"`                 //*493 SUM percentage of challenges failed by a particular blobber
	AllocatedStorage     int64 `json:"allocated_storage"`                //*490 SUM clients have locked up storage by purchasing allocations (new + previous + update -sub fin+cancel or reduceed)
	MaxCapacityStorage   int64 `json:"max_capacity_storage"`             //*491 SUM all storage from blobber settings
	StakedStorage        int64 `json:"staked_storage"`                   //*491 SUM staked capacity by delegates
	UsedStorage          int64 `json:"used_storage"`                     //*491 SUM this is the actual usage or data that is in the server - write markers (triggers challenge pool / the price).(bytes written used capacity)
	TransactionsCount    int64 `json:"transactions_count"`               // Total number of transactions in a block
	UniqueAddresses      int64 `json:"unique_addresses"`                 // Total unique address
	BlockCount           int64 `json:"block_count"`                      // Total number of blocks currently
	AverageTxnFee        int64 `json:"avg_txn_fee"`                      // Average transaction fee per block
	CreatedAt            int64 `gorm:"autoCreateTime" json:"created_at"` // Snapshot creation date
	BlobberCount		 int64 `json:"blobber_count"`                    // Total number of blobbers
	MinerCount			 int64 `json:"miner_count"`                      // Total number of miners
	SharderCount		 int64 `json:"sharder_count"`                    // Total number of sharders
	ValidatorCount		 int64 `json:"validator_count"`                  // Total number of validators
	AuthorizerCount		 int64 `json:"authorizer_count"`                  // Total number of authorizers
}

func (s *Snapshot) providerCount(provider spenum.Provider) int64 {
	switch provider {
	case spenum.Blobber:
		return s.BlobberCount
	case spenum.Miner:
		return s.MinerCount
	case spenum.Sharder:
		return s.SharderCount
	case spenum.Validator:
		return s.ValidatorCount
	case spenum.Authorizer:
		return s.AuthorizerCount
	default:
		return 0
	}
}

// updateAveragesAfterIncrement updates average fields before incrementing the count of the provider.
func (s *Snapshot) updateAveragesBeforeIncrement(provider spenum.Provider) {
	providerCount := s.providerCount(provider)
	if providerCount > 0 {
		s.AverageWritePrice = (s.AverageWritePrice * providerCount) / (providerCount + 1)
	}
}

// updateAveragesAfterDecrement updates average fields after decrementing the count of the provider.
func (s *Snapshot) updateAveragesBeforeDecrement(provider spenum.Provider) {
	providerCount := s.providerCount(provider)
	if providerCount > 0 {
		s.AverageWritePrice = (s.AverageWritePrice * providerCount) / (providerCount - 1)
	}
}

// ApplyDiff applies diff values of global snapshot fields to the current snapshot according to each field's update formula.
// For some fields, the count of the providers may be needed so a provider parameter is added.
func (s *Snapshot) ApplyDiff(diff *Snapshot, provider spenum.Provider) {
	logging.Logger.Debug("SnapshotDiff", zap.Any("provider", provider), zap.Any("diff", diff), zap.Any("snapshot_before", s))
	s.TotalMint += diff.TotalMint
	s.TotalChallengePools += diff.TotalChallengePools
	s.ActiveAllocatedDelta += diff.ActiveAllocatedDelta
	s.ZCNSupply += diff.ZCNSupply
	s.TotalValueLocked += diff.TotalValueLocked
	s.ClientLocks += diff.ClientLocks
	s.MinedTotal += diff.MinedTotal
	s.TotalStaked += diff.TotalStaked
	s.TotalRewards += diff.TotalRewards
	s.SuccessfulChallenges += diff.SuccessfulChallenges
	s.TotalChallenges += diff.TotalChallenges
	s.AllocatedStorage += diff.AllocatedStorage
	s.MaxCapacityStorage += diff.MaxCapacityStorage
	s.StakedStorage += diff.StakedStorage
	s.UsedStorage += diff.UsedStorage
	s.TransactionsCount += diff.TransactionsCount
	s.UniqueAddresses += diff.UniqueAddresses
	s.BlockCount += diff.BlockCount

	if s.TransactionsCount > 0 {
		s.AverageTxnFee += diff.AverageTxnFee / s.TransactionsCount
	}

	providerCount := s.providerCount(provider)
	if providerCount > 0 {
		s.AverageWritePrice += diff.AverageWritePrice / providerCount
	}

	logging.Logger.Debug("SnapshotDiff", zap.Any("snapshot_after", s))
}

type FieldType int

const (
	Allocated = iota
	MaxCapacity
	Staked
	Used
)

type AllocationValueChanged struct {
	FieldType    FieldType
	AllocationId string
	Delta        int64
}
type AllocationBlobberValueChanged struct {
	FieldType    FieldType
	AllocationId string
	BlobberId    string
	Delta        int64
}

func (edb *EventDb) ReplicateSnapshots(round int64, limit int) ([]Snapshot, error) {
	var snapshots []Snapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM snapshots WHERE round > ? ORDER BY round LIMIT ?", round, limit).Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}
	return snapshots, nil
}

func (edb *EventDb) addSnapshot(s Snapshot) error {
	return edb.Store.Get().Create(&s).Error
}

func (edb *EventDb) GetGlobal() (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Order("round desc").First(&s)
	return s, res.Error
}

func (gs *Snapshot) update(e []Event) {
	for _, event := range e {
		logging.Logger.Debug("update snapshot",
			zap.String("tag", event.Tag.String()),
			zap.Int64("block_number", event.BlockNumber))
		switch event.Tag {
		case TagToChallengePool:
			cp, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalChallengePools += cp.Amount
		case TagFromChallengePool:
			cp, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalChallengePools -= cp.Amount
		case TagAddMint:
			m, ok := fromEvent[state.Mint](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalMint += int64(m.Amount)
			gs.ZCNSupply += int64(m.Amount)
			logging.Logger.Info("snapshot update TagAddMint",
				zap.Int64("total_mint", gs.TotalMint), zap.Int64("zcn_supply", gs.ZCNSupply))
		case TagBurn:
			m, ok := fromEvent[state.Burn](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ZCNSupply -= int64(m.Amount)
			logging.Logger.Info("snapshot update TagBurn",
				zap.Int64("zcn_supply", gs.ZCNSupply))
		case TagLockStakePool:
			ds, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}

			var total int64
			for _, d := range *ds {
				total += d.Amount
				gs.TotalValueLocked += d.Amount
				gs.TotalStaked += d.Amount
			}
			logging.Logger.Debug("update lock stake pool", zap.Int64("round", event.BlockNumber), zap.Int64("amount", total),
				zap.Int64("total_amount", gs.TotalValueLocked))
		case TagUnlockStakePool:
			ds, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.TotalValueLocked -= d.Amount
				gs.TotalStaked -= d.Amount
			}
		case TagLockWritePool:
			ds, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks += d.Amount
				gs.TotalValueLocked += d.Amount
			}
		case TagUnlockWritePool:
			ds, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks -= d.Amount
				gs.TotalValueLocked -= d.Amount
			}
		case TagLockReadPool:
			ds, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks += d.Amount
				gs.TotalValueLocked += d.Amount
			}
		case TagUnlockReadPool:
			ds, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks -= d.Amount
				gs.TotalValueLocked -= d.Amount
			}
		case TagStakePoolReward:
			spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, spu := range *spus {
				for _, r := range spu.DelegateRewards {
					dr, err := r.Int64()
					if err != nil {
						logging.Logger.Error("snapshot",
							zap.Any("event", event.Data), zap.Error(err))
						continue
					}
					gs.MinedTotal += dr
				}
			}
		case TagFinalizeBlock:
			block, ok := fromEvent[Block](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TransactionsCount += int64(block.NumTxns)
			gs.BlockCount += 1
		case TagUniqueAddress:
			gs.UniqueAddresses += 1
		case TagAddTransactions:
			txns, ok := fromEvent[[]Transaction](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			averageFee := 0
			for _, txn := range *txns {
				averageFee += int(txn.Fee)
			}
			averageFee = averageFee / len(*txns)
			gs.AverageTxnFee = int64(averageFee)
		case TagAddBlobber:
			gs.updateAveragesBeforeIncrement(spenum.Blobber)
			gs.BlobberCount += 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "AddBlobber"), zap.Any("snapshot", gs))
		case TagDeleteBlobber:
			gs.updateAveragesBeforeDecrement(spenum.Blobber)
			gs.BlobberCount -= 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "DeleteBlobber"), zap.Any("snapshot", gs))
		case TagAddAuthorizer:
			gs.AuthorizerCount += 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "AddAuthorizer"), zap.Any("snapshot", gs))
		case TagDeleteAuthorizer:
			gs.AuthorizerCount -= 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "DeleteAuthorizer"), zap.Any("snapshot", gs))
		case TagAddMiner:
			gs.MinerCount += 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "AddMiner"), zap.Any("snapshot", gs))
		case TagDeleteMiner:
			gs.MinerCount -= 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "DeleteMiner"), zap.Any("snapshot", gs))
		case TagAddSharder:
			gs.SharderCount += 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "AddSharder"), zap.Any("snapshot", gs))
		case TagDeleteSharder:
			gs.SharderCount -= 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "DeleteSharder"), zap.Any("snapshot", gs))
		case TagAddOrOverwiteValidator:
			gs.ValidatorCount += 1
			logging.Logger.Debug("SnapshotProvider", zap.String("type", "AddValidator"), zap.Any("snapshot", gs))
		}

	}
}
