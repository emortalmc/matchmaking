package mmf

import (
	"github.com/google/uuid"
	"matchmaker/pkg/common/modeprofile"
	"open-match.dev/open-match/pkg/pb"
)

// MakeInstantMatches
// Immediately returns a match with all tickets in the pool
// but groups them together to reduce allocations
func MakeInstantMatches(profile modeprofile.ModeProfile, tickets []*pb.Ticket) ([]*pb.Match, error) {
	if len(tickets) <= profile.MinPlayers {
		return nil, nil
	}

	var matches []*pb.Match
	for len(tickets) >= profile.MinPlayers {
		var matchTickets []*pb.Ticket
		for i := 0; i < profile.MaxPlayers && i < len(tickets); i++ {
			ticket := tickets[0]
			// Remove the Tickets from this pool and add to the match proposal.
			matchTickets = append(matchTickets, ticket)
			tickets = tickets[1:]
		}

		match := newMatch(uuid.New(), profile, matchTickets)
		matches = append(matches, match)
	}

	return matches, nil
}
