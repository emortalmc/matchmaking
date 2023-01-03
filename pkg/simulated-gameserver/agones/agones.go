package agones

import (
	sdk2 "agones.dev/agones/pkg/sdk"
	sdk "agones.dev/agones/sdks/go"
	"context"
	"fmt"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/strings/slices"
	"matchmaker/pkg/simulated-gameserver/config"
	"strconv"
	"time"
)

const (
	maxRunningGames = 10
)

var (
	Sdk, _    = sdk.NewSDK()
	logger, _ = zap.NewDevelopment()
	SelfGs, _ = Sdk.GameServer()

	lastAllocated   time.Time
	RunningMatchIds []string                  // list of match ids on this gameserver
	BackfillIds     = make(map[string]string) // maps a match ID to its backfill ID
)

func UpdateShouldAllocate() {
	shouldAllocate := len(RunningMatchIds) < maxRunningGames
	err := Sdk.SetLabel("should-allocate", strconv.FormatBool(shouldAllocate)) // translates to agonesSdk.dev/sdk-should-allocate
	if err != nil {
		logger.Error("Failed to updateShouldAllocate should allocate label", zap.Bool("shouldAllocate", shouldAllocate), zap.Error(err))
	}
}

func StartPlayerTrackingIfEnabled() {
	logger.Info("Attempting to start player tracking", zap.Bool("enabled", *config.EnablePlayerTracking), zap.Int64("capacity", *config.PlayerTrackingSlots))
	if !*config.EnablePlayerTracking {
		return
	}
	err := Sdk.Alpha().SetPlayerCapacity(*config.PlayerTrackingSlots)
	if err != nil {
		logger.Error("Could not set player capacity", zap.Error(err))
	} else {
		logger.Info("Set player capacity", zap.Int("capacity", int(*config.PlayerTrackingSlots)))
	}
}

// DoHealth sends the regular Health Pings
// Taken from example Agones Go gameserver
func DoHealth() {
	ctx := context.Background()
	tick := time.Tick(10 * time.Second)
	for {
		logger.Debug("Health Ping")
		err := Sdk.Health()
		if err != nil {
			logger.Error("Could not send health ping", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			logger.Debug("Stopped health pings")
			return
		case <-tick:
		}
	}
}

func IsNewAllocation(gs *sdk2.GameServer) bool {
	str := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
	if str == "" {
		return false
	}
	ts, err := time.Parse(time.RFC3339, str)
	if err != nil {
		logger.Error("Could not parse last-allocated annotation", zap.Error(err))
		return false
	}
	if ts.After(lastAllocated) {
		lastAllocated = ts
		return true
	}
	return false
}

// IsBackfill checks if the match ID of the allocation is already
// running on this GameServer. If so, the match is a backfill
func IsBackfill(allocation Allocation) bool {
	return slices.Contains(RunningMatchIds, allocation.MatchId)
}

type Allocation struct {
	MatchId         string   `json:"matchId"`
	ExpectedPlayers []string `json:"expectedPlayers"`
	BackfillId      string   `json:"backfillId"`
}

func ParseAllocation(gs *sdk2.GameServer) (Allocation, error) {
	annotations := gs.ObjectMeta.Annotations

	var matchId string
	var playerIds []string
	var backfillId string
	for k, v := range annotations {
		if k == "openmatch.dev/match-id" {
			matchId = v
			continue
		}
		if k == "openmatch.dev/backfill-id" {
			backfillId = v
			continue
		}
		if k == "openmatch.dev/expected-players" {
			err := json.Unmarshal([]byte(v), &playerIds)
			if err != nil {
				return Allocation{}, fmt.Errorf("could not parse player-ids (%s) annotation: %w", v, err)
			}
			continue
		}
	}

	return Allocation{
		MatchId:         matchId,
		ExpectedPlayers: playerIds,
		BackfillId:      backfillId,
	}, nil
}

func TrackPlayersOnAgones(allocation Allocation) {
	if !*config.EnablePlayerTracking {
		return
	}
	for _, pId := range allocation.ExpectedPlayers {
		_, err := Sdk.Alpha().PlayerConnect(pId)
		if err != nil {
			logger.Error("Could not track player on Agones", zap.String("playerId", pId), zap.Error(err))
		}
	}
}
