package tracker

// MatchPlayers is a map of match IDs to player IDs
var MatchPlayers = make(map[string][]string)

func AddPlayers(matchId string, playerIds ...string) {
	MatchPlayers[matchId] = append(MatchPlayers[matchId], playerIds...)
}
