package mmf

import (
	"github.com/google/uuid"
	"log"
	"matchmaker/pkg/common/modeprofile"
	"matchmaker/pkg/common/notifier"
	"matchmaker/pkg/common/utils"
	"open-match.dev/open-match/pkg/pb"
	"time"
)

// MakeCountdownMatches
// Fill any backfills with current tickets in the pool
// Create new matches for any remaining tickets
// Create backfills for created matches that aren't full
// todo: we should return fewer errors, log them and handle gracefully to avoid 'freezes' in matchmaking on an error.

var (
	// countdowns map[poolName]teleportTime
	countdowns = make(map[string]time.Time)
	// countdownPlayers map[poolName][]playerId
	countdownPlayers = make(map[string][]string)
)

// MakeCountdownMatches
// if ticket count >= max players, create a match
// if ticket count < min players, stop
// if ticket count < max players, create a countdown until the match is made anyway
// if ticket count < max players and countdown is over, create a match
func MakeCountdownMatches(profile modeprofile.ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) ([]*pb.Match, error) {
	// update countdownPlayers. We need to update this every time, because players can leave the pool.
	countdownPlayers[pool.Name] = utils.ExtractPlayerIdsFromTickets(tickets)

	if len(tickets) == 0 {
		return nil, nil
	}

	countdown, hasCountdown := countdowns[pool.Name]

	// delete countdown if players have left the queue
	if hasCountdown && len(tickets) < profile.MinPlayers {
		delete(countdowns, pool.Name)
		notifier.NotifyPlayersOfCancelledCountdown(countdownPlayers[pool.Name])
	}

	var matches []*pb.Match
	// create full matches
	if len(tickets) >= profile.MaxPlayers {
		madeMatches, newTickets := makeFullMatches(profile, tickets)
		tickets = newTickets
		matches = madeMatches

		// delete the countdown as we have made a match
		// there is no need to notify here as the server is notified by the director when a gameserver is assigned instead
		delete(countdowns, pool.Name)

		log.Printf("makeFullMatches done: tickets: %d", len(tickets))
	}

	// no more matches or a countdown can be made, return
	if len(tickets) < profile.MinPlayers {
		return matches, nil
	}

	if hasCountdown && time.Now().After(countdown) {
		// create a match
		match := newMatch(uuid.New(), profile, tickets)

		matches = append(matches, match)

		// delete the countdown as we have made a match
		// there is no need to notify here as the server is notified by the director when a gameserver is assigned instead
		delete(countdowns, pool.Name)
		return matches, nil
	}

	// still tickets left, create a countdown
	if len(tickets) >= profile.MinPlayers {
		if !hasCountdown {
			countdowns[pool.Name] = time.Now().Add(10 * time.Second)
		}
		// notify players of the countdown
		notifier.NotifyPlayersOfPendingMatch(countdownPlayers[pool.Name], countdowns[pool.Name])
	}

	log.Printf("MakeCountdownMatches finished: tickets: %d", len(tickets))

	return matches, nil
}

//// handleBackfills fills current backfills with tickets in the pool
//// and updates the backfill with: the new tickets, the new open slots
//func handleBackfills(profile modeprofile.ModeProfile, tickets []*pb.Ticket, backfills []*pb.Backfill) ([]*pb.Match, []*pb.Ticket, []*pb.Backfill, error) {
//	var matches []*pb.Match
//	for _, backfill := range backfills {
//		slots, err := getBackfillSlots(backfill)
//		if err != nil {
//			return nil, tickets, backfills, err
//		}
//		if slots == 0 {
//			continue
//		}
//		// fill the backfill with tickets from the tickets array
//		// remove tickets from the tickets array
//		// update the open slots of the backfill
//		var matchTickets []*pb.Ticket
//		for slots > 0 && len(tickets) > 0 {
//			matchTickets = append(matchTickets, tickets[0])
//			tickets = tickets[1:]
//			slots--
//		}
//
//		// the backfill has been updated. we must update the available slots
//		// and create a Match to send players to the game server
//		if len(matchTickets) > 0 {
//			err := setBackfillSlots(backfill, slots)
//			if err != nil {
//				return matches, tickets, backfills, err
//			}
//			match := newMatch(uuid.New(), profile, matchTickets, backfill)
//			matches = append(matches, match)
//		}
//
//	}
//
//	return matches, tickets, backfills, nil
//}

// makeFullMatches creates full matches from tickets in the pool.
// returns: creates matches, remaining tickets that are unused.
func makeFullMatches(profile modeprofile.ModeProfile, tickets []*pb.Ticket) ([]*pb.Match, []*pb.Ticket) {
	var matches []*pb.Match
	for len(tickets) >= profile.MaxPlayers {
		var matchTickets []*pb.Ticket
		for i := 0; i < profile.MaxPlayers; i++ {
			ticket := tickets[0]
			// Remove the Tickets from this pool and add to the match proposal.
			matchTickets = append(matchTickets, ticket)
			tickets = tickets[1:]
		}

		matches = append(matches, newMatch(uuid.New(), profile, matchTickets))
	}

	log.Printf("makeFullMatches finished: tickets: %d", len(tickets))
	return matches, tickets
}

//// makeMatchWithBackfill makes not full match, creates backfill for it with openSlots = maxPlayersPerMatch-len(tickets).
//func makeMatchWithBackfill(profile modeprofile.ModeProfile, pool *pb.Pool, tickets []*pb.Ticket) (*pb.Match, error) {
//	if len(tickets) == 0 {
//		return nil, fmt.Errorf("tickets are required")
//	}
//
//	if len(tickets) >= profile.MaxPlayers {
//		return nil, fmt.Errorf("too many tickets")
//	}
//
//	matchId := uuid.New()
//	backfill, err := newBackfill(matchId, pool, int32(profile.MaxPlayers-len(tickets)))
//	if err != nil {
//		return nil, err
//	}
//
//	match := newMatch(matchId, profile, tickets, backfill)
//	// indicates that it is a new match and new game server should be allocated for it
//	match.AllocateGameserver = true
//
//	return match, nil
//}

func newMatch(id uuid.UUID, profile modeprofile.ModeProfile, tickets []*pb.Ticket) *pb.Match {
	match := &pb.Match{
		MatchId:            id.String(),
		MatchProfile:       profile.Name,
		MatchFunction:      profile.Name,
		Tickets:            tickets,
		AllocateGameserver: true,
	}

	return match
}

//func newBackfill(matchId uuid.UUID, pool *pb.Pool, slots int32) (*pb.Backfill, error) {
//	if slots <= 0 {
//		return nil, fmt.Errorf("slots must be greater than 0")
//	}
//
//	originalMatchId, err := anypb.New(wrapperspb.String(matchId.String()))
//	if err != nil {
//		return nil, err
//	}
//	searchFields := newSearchFields(pool)
//	backfill := &pb.Backfill{
//		SearchFields: searchFields,
//		Extensions: map[string]*anypb.Any{
//			"originalMatchId": originalMatchId,
//		},
//	}
//
//	err = setBackfillSlots(backfill, slots)
//	if err != nil {
//		return nil, err
//	}
//
//	return backfill, nil
//}
