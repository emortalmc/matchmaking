package main

import (
	sdk2 "agones.dev/agones/pkg/sdk"
	"go.uber.org/zap"
	"matchmaker/pkg/simulated-gameserver/agones"
	"matchmaker/pkg/simulated-gameserver/config"
	"matchmaker/pkg/simulated-gameserver/openmatch"
	"matchmaker/pkg/simulated-gameserver/tracker"
)

var (
	logger, _ = zap.NewDevelopment()
)

// todo we need to clean up things like the backfill ID in case they aren't set for a game
// but are for a subsequent game.
// todo logic for player tracking and toggleable high density mode
func main() {
	logger.Info("Starting simulated gameserver")
	config.Init()

	go agones.DoHealth()

	// Get self details
	gs, err := agones.Sdk.GameServer()
	if err != nil {
		logger.Error("Could not get gameserver details", zap.Error(err))
		return
	}
	logger.Debug("Got gameserver details", zap.Any("gameserver", gs))

	err = agones.Sdk.Ready()
	if err != nil {
		logger.Error("Could not send ready", zap.Error(err))
		return
	}

	err = agones.Sdk.WatchGameServer(gameServerChange)
	if err != nil {
		logger.Error("Could not watch gameserver", zap.Error(err))
		return
	}

	agones.StartPlayerTrackingIfEnabled()

	// Wait forever
	select {}
}

func gameServerChange(gs *sdk2.GameServer) {
	logger.Debug("GameServer changed", zap.Any("gameserver", gs))
	if agones.IsNewAllocation(gs) {
		allocation, err := agones.ParseAllocation(gs)
		if err != nil {
			logger.Error("Could not parse new allocation", zap.Error(err))
			return
		}
		logger.Info("Allocation", zap.Any("allocation", allocation))

		trackPlayers(allocation)

		agones.RunningMatchIds = append(agones.RunningMatchIds, allocation.MatchId)
		if allocation.BackfillId != "" {
			agones.BackfillIds[allocation.MatchId] = allocation.BackfillId
			openmatch.RegisterBackfill(allocation.BackfillId)
		}

		// updates the label of whether this gameserver can take more allocations
		// and be packed more.
		agones.UpdateShouldAllocate()
		agones.TrackPlayersOnAgones(allocation)

		openmatch.HandlePossibleUpdate(allocation)

		logger.Debug("Players now tracked", zap.Any("players", tracker.MatchPlayers))
	}
}

// trackPlayers adds players to their match if backfilling or creates a new match in the map if a new allocation.
func trackPlayers(allocation agones.Allocation) {
	if agones.IsBackfill(allocation) {
		tracker.AddPlayers(allocation.MatchId, allocation.ExpectedPlayers...)
	} else {
		tracker.MatchPlayers[allocation.MatchId] = allocation.ExpectedPlayers
	}
}
