// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"fmt"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/utils/units"
)

func TestRewardLongerDurationBonus(t *testing.T) {
	shortDuration := 14 * 24 * time.Hour
	totalDuration := 365 * 24 * time.Hour
	shortBalance := units.KiloDjtx
	for i := 0; i < int(totalDuration/shortDuration); i++ {
		reward := Reward(shortDuration, shortBalance, 359*units.MegaDjtx+shortBalance)
		shortBalance += reward
	}
	reward := Reward(totalDuration%shortDuration, shortBalance, 359*units.MegaDjtx+shortBalance)
	shortBalance += reward

	longBalance := units.KiloDjtx
	longBalance += Reward(totalDuration, longBalance, 359*units.MegaDjtx+longBalance)

	if shortBalance >= longBalance {
		t.Fatalf("should promote stakers to stake longer")
	}
}

func TestRewards(t *testing.T) {
	tests := []struct {
		duration       time.Duration
		stakeAmount    uint64
		existingAmount uint64
		expectedReward uint64
	}{
		// Max duration:
		{ // (720M - 360M) * (1M / 360M) * 12%
			duration:       MaximumStakingDuration,
			stakeAmount:    units.MegaDjtx,
			existingAmount: 360 * units.MegaDjtx,
			expectedReward: 120 * units.KiloDjtx,
		},
		{ // (720M - 400M) * (1M / 400M) * 12%
			duration:       MaximumStakingDuration,
			stakeAmount:    units.MegaDjtx,
			existingAmount: 400 * units.MegaDjtx,
			expectedReward: 96 * units.KiloDjtx,
		},
		{ // (720M - 400M) * (2M / 400M) * 12%
			duration:       MaximumStakingDuration,
			stakeAmount:    2 * units.MegaDjtx,
			existingAmount: 400 * units.MegaDjtx,
			expectedReward: 192 * units.KiloDjtx,
		},
		{ // (720M - 720M) * (1M / 720M) * 12%
			duration:       MaximumStakingDuration,
			stakeAmount:    units.MegaDjtx,
			existingAmount: SupplyCap,
			expectedReward: 0,
		},
		// Min duration:
		// (720M - 360M) * (1M / 360M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       MinimumStakingDuration,
			stakeAmount:    units.MegaDjtx,
			existingAmount: 360 * units.MegaDjtx,
			expectedReward: 274122724713,
		},
		// (720M - 360M) * (.005 / 360M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       MinimumStakingDuration,
			stakeAmount:    minStake,
			existingAmount: 360 * units.MegaDjtx,
			expectedReward: 1370,
		},
		// (720M - 400M) * (1M / 400M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       MinimumStakingDuration,
			stakeAmount:    units.MegaDjtx,
			existingAmount: 400 * units.MegaDjtx,
			expectedReward: 219298179771,
		},
		// (720M - 400M) * (2M / 400M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       MinimumStakingDuration,
			stakeAmount:    2 * units.MegaDjtx,
			existingAmount: 400 * units.MegaDjtx,
			expectedReward: 438596359542,
		},
		// (720M - 720M) * (1M / 720M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       MinimumStakingDuration,
			stakeAmount:    units.MegaDjtx,
			existingAmount: SupplyCap,
			expectedReward: 0,
		},
	}
	for _, test := range tests {
		name := fmt.Sprintf("reward(%s,%d,%d)==%d",
			test.duration,
			test.stakeAmount,
			test.existingAmount,
			test.expectedReward,
		)
		t.Run(name, func(t *testing.T) {
			reward := Reward(
				test.duration,
				test.stakeAmount,
				test.existingAmount,
			)
			if reward != test.expectedReward {
				t.Fatalf("expected %d; got %d", test.expectedReward, reward)
			}
		})
	}
}
