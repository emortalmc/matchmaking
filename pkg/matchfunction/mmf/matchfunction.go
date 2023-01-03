package mmf

import (
	"fmt"
	"log"
	"matchmaker/pkg/common/modeprofile"
	"matchmaker/pkg/common/modeprofile/config"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
	"strings"
)

const (
	poolName = "all"
)

// Run is this match function's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *MatchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.
	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())
	matchProfile := req.GetProfile()
	modeProfile, err := config.GetModeProfileByMatchProfileName(matchProfile.GetName())
	if err != nil {
		err := fmt.Errorf("match profile %v not supported", req.GetProfile().GetName())
		log.Printf(err.Error())
		return err
	}

	poolTickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, req.GetProfile().GetPools())
	if err != nil {
		log.Printf("Failed to query tickets for the given pools, got %s", err.Error())
		return err
	}

	//poolBackfills, err := matchfunction.QueryBackfillPools(stream.Context(), s.queryServiceClient, req.GetProfile().GetPools())
	//if err != nil {
	//	log.Printf("Failed to query backfills for the given pools, got %s", err.Error())
	//	return err
	//}

	ticketCount := getTicketCount(poolTickets)
	//backfillCount := getBackfillCount(poolBackfills)
	log.Printf("Got %v tickets for pools [%v]", ticketCount, strings.Join(getPoolNames(req.GetProfile().GetPools()), ", "))
	log.Printf("Tickets: %v", poolTickets)

	// Generate proposals.
	proposals, err := makeMatches(modeProfile, makePoolMap(req.Profile.Pools), poolTickets)
	if err != nil {
		log.Printf("Failed to generate matches, got %s", err.Error())
		return err
	}

	log.Printf("Streaming %v proposals to Open Match", len(proposals))
	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			log.Printf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}

func makePoolMap(pools []*pb.Pool) map[string]*pb.Pool {
	poolMap := make(map[string]*pb.Pool)
	for _, pool := range pools {
		poolMap[pool.GetName()] = pool
	}
	return poolMap
}

func makeMatches(modeProfile modeprofile.ModeProfile, pools map[string]*pb.Pool, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	pool := pools[poolName]
	tickets := poolTickets[poolName]

	if len(tickets) == 0 {
		return nil, nil
	}

	matches, err := modeProfile.MatchFunction(modeProfile, pool, tickets)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func getPoolNames(pools []*pb.Pool) []string {
	var poolNames []string
	for _, pool := range pools {
		poolNames = append(poolNames, pool.GetName())
	}
	return poolNames
}

func getTicketCount(ticketMap map[string][]*pb.Ticket) int {
	count := 0
	for _, tickets := range ticketMap {
		count += len(tickets)
	}
	return count
}

func getBackfillCount(backfillMap map[string][]*pb.Backfill) int {
	count := 0
	for _, backfills := range backfillMap {
		count += len(backfills)
	}
	return count
}
