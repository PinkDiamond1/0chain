package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

func (sc *StorageSmartContract) newWrite(statectx c_state.StateContextI, writeSize int64) error {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	switch err {
	case nil, util.ErrValueNotPresent:
		stats.Stats.NumWrites++
		stats.Stats.UsedSize += writeSize
		_, err = statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
		return err
	default:
		return err
	}
}

func (sc *StorageSmartContract) newRead(statectx c_state.StateContextI, readSize int64) error {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}

	stats.Stats.ReadsSize += readSize
	_, err = statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
	return err
}

func (sc *StorageSmartContract) newChallenge(
	statectx c_state.StateContextI,
	challengeTimestamp common.Timestamp,
	challenges int64) error {

	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}

	stats.Stats.OpenChallenges += challenges
	stats.Stats.TotalChallenges += challenges
	stats.LastChallengedSize = stats.Stats.UsedSize
	stats.LastChallengedTime = challengeTimestamp
	_, err = statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
	return err
}

func (sc *StorageSmartContract) challengeResolved(statectx c_state.StateContextI, challengedPassed bool) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil {
		return
	}

	stats.Stats.OpenChallenges--
	if challengedPassed {
		stats.Stats.SuccessChallenges++
	} else {
		stats.Stats.FailedChallenges++
	}
	statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
}
