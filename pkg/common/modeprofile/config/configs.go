package config

import (
	v1 "agones.dev/agones/pkg/apis/allocation/v1"
	"fmt"
	"matchmaker/pkg/common/matchprofile"
	"matchmaker/pkg/common/mmf"
	"matchmaker/pkg/common/modeprofile"
	"matchmaker/pkg/common/selector"
	"open-match.dev/open-match/pkg/pb"
)

var ModeProfiles = map[string]modeprofile.ModeProfile{
	"marathon": {
		Name:       "marathon",
		PoolName:   "marathon",
		FleetName:  "marathon",
		MinPlayers: 1,
		MaxPlayers: 100, // The server knows this is a singleplayer game, we can reduce load by using backfilling.
		Selector: func(profile modeprofile.ModeProfile, match *pb.Match) *v1.GameServerAllocation {
			return selector.CommonPlayerBasedSelector(profile, match, int64(len(match.Tickets)))
		},
		MatchProfile: matchprofile.CommonProfile("marathon", "marathon"),
		MatchFunction: func(profile modeprofile.ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) ([]*pb.Match, error) {
			return mmf.MakeInstantMatches(profile, tickets)
		},
	},
	"lobby": {
		Name:       "lobby",
		PoolName:   "lobby",
		FleetName:  "lobby",
		MinPlayers: 1,
		MaxPlayers: 50,
		Selector: func(profile modeprofile.ModeProfile, match *pb.Match) *v1.GameServerAllocation {
			return selector.CommonPlayerBasedSelector(profile, match, int64(len(match.Tickets)))
		},
		MatchProfile: matchprofile.CommonProfile("lobby", "lobby"),
		MatchFunction: func(profile modeprofile.ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) ([]*pb.Match, error) {
			return mmf.MakeInstantMatches(profile, tickets)
		},
	},
	"block_sumo": {
		Name:       "block_sumo",
		PoolName:   "block_sumo",
		FleetName:  "block-sumo",
		MinPlayers: 2,
		MaxPlayers: 12,
		Selector: func(profile modeprofile.ModeProfile, match *pb.Match) *v1.GameServerAllocation {
			return selector.CommonSelector(profile, match)
		},
		MatchProfile: matchprofile.CommonProfile("block_sumo", "block_sumo"),
		MatchFunction: func(profile modeprofile.ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) ([]*pb.Match, error) {
			return mmf.MakeCountdownMatches(profile, pool, tickets)
		},
	},
	"minesweeper": {
		Name:       "minesweeper",
		PoolName:   "minesweeper",
		FleetName:  "minesweeper",
		MinPlayers: 1,
		MaxPlayers: 5,
		Selector: func(profile modeprofile.ModeProfile, match *pb.Match) *v1.GameServerAllocation {
			return selector.CommonPlayerBasedSelector(profile, match, int64(len(match.Tickets)))
		},
		MatchProfile: matchprofile.CommonProfile("minesweeper", "minesweeper"),
		MatchFunction: func(profile modeprofile.ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) ([]*pb.Match, error) {
			return mmf.MakeInstantMatches(profile, tickets)
		},
	},
}

func GetModeProfileByMatchProfileName(name string) (modeprofile.ModeProfile, error) {
	for _, profile := range ModeProfiles {
		if profile.MatchProfile.Name == name {
			return profile, nil
		}
	}
	return modeprofile.ModeProfile{}, fmt.Errorf("no mode profile found for match profile %s", name)
}
