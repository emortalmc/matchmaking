package modeprofile

import (
	v1 "agones.dev/agones/pkg/apis/allocation/v1"
	"open-match.dev/open-match/pkg/pb"
)

type ModeProfile struct {
	Name      string `json:"name"`
	PoolName  string `json:"poolName"`
	FleetName string `json:"fleetName"`
	//TeamSize   int // currently unused but can be used for parties later.

	Selector     func(profile ModeProfile, match *pb.Match) *v1.GameServerAllocation `json:"-"`
	MatchProfile *pb.MatchProfile                                                    `json:"matchProfile"`

	MinPlayers    int                                                                                 `json:"minPlayers"`
	MaxPlayers    int                                                                                 `json:"maxPlayers"`
	UseCountdown  bool                                                                                `json:"useCountdown"`
	MatchFunction func(profile ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) ([]*pb.Match, error) `json:"-"`
}
