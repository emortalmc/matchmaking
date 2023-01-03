package main

import (
	v1 "agones.dev/agones/pkg/apis/allocation/v1"
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"matchmaker/pkg/common/modeprofile"
	"matchmaker/pkg/common/modeprofile/config"
	"matchmaker/pkg/common/notifier"
	"matchmaker/pkg/common/utils/kubernetes"
	"sync"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

const (
	Namespace = "towerdefence"

	// The endpoint for the Open Match Backend service.
	omBackendEndpoint = "open-match-backend.open-match.svc:50505"
	// The Host and Port for the Match Function service endpoint.
	functionHostName       = "matchfunction.towerdefence.svc"
	functionPort     int32 = 50502

	minTimeBetweenRuns = 1000 * time.Millisecond
)

var (
	logger, _ = zap.NewProduction()
)

func main() {
	// Connect to OM Backend.
	conn, err := grpc.Dial(omBackendEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)

	if err != nil {
		logger.Error("Failed to connect to Open Match Backend, got %s", zap.Error(err))
	}

	defer conn.Close()
	be := pb.NewBackendServiceClient(conn)

	modeProfiles := config.ModeProfiles

	logger.Info("Fetching matches for profiles",
		zap.Int("profileCount", len(modeProfiles)),
		zap.Any("profiles", modeProfiles),
	)

	// Only run every x milliseconds, but if that has already passed, run immediately.
	for {
		lastRunTime := time.Now()
		run(be, modeProfiles)
		timeSinceLastRun := time.Since(lastRunTime)
		if timeSinceLastRun < minTimeBetweenRuns {
			time.Sleep(minTimeBetweenRuns - timeSinceLastRun)
		}
	}
}

func run(be pb.BackendServiceClient, profiles map[string]modeprofile.ModeProfile) {
	var wg sync.WaitGroup
	for _, p := range profiles {
		wg.Add(1)
		go func(wg *sync.WaitGroup, p modeprofile.ModeProfile) {
			defer wg.Done()
			matches, err := fetch(be, p.MatchProfile)
			if err != nil {
				logger.Info("Failed to fetch matches", zap.String("profileName", p.Name), zap.Error(err))
				return
			}

			logger.Info("Generated matches", zap.Int("generated", len(matches)), zap.String("profileName", p.Name))
			if err := assign(be, p, matches); err != nil {
				logger.Error("Failed to assign servers to matches", zap.Error(err))
				return
			}
		}(&wg, p)
	}

	wg.Wait()
}

func fetch(be pb.BackendServiceClient, p *pb.MatchProfile) ([]*pb.Match, error) {
	req := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{
			Host: functionHostName,
			Port: functionPort,
			Type: pb.FunctionConfig_GRPC,
		},
		Profile: p,
	}

	stream, err := be.FetchMatches(context.Background(), req)
	if err != nil {
		return nil, err
	}

	var result []*pb.Match
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		result = append(result, resp.GetMatch())
	}

	return result, nil
}

func assign(be pb.BackendServiceClient, profile modeprofile.ModeProfile, matches []*pb.Match) error {
	for _, match := range matches {
		if !match.GetAllocateGameserver() {
			continue
		}
		var ticketIDs []string
		for _, t := range match.GetTickets() {
			ticketIDs = append(ticketIDs, t.Id)
		}

		// Request an allocation based on that defined in the ModeProfile
		allocation, err := kubernetes.AgonesClient.AllocationV1().
			GameServerAllocations(Namespace).
			Create(context.Background(), profile.Selector(profile, match), v12.CreateOptions{})

		if err != nil {
			logger.Error("Failed to allocate server", zap.String("matchId", match.MatchId), zap.Error(err))
			continue
		}
		status := allocation.Status
		if status.State != v1.GameServerAllocationAllocated {
			logger.Error("Failed to allocate server", zap.String("matchId", match.MatchId), zap.String("state", string(status.State)))
			continue
		}
		conn := fmt.Sprintf("%s:%d", status.Address, status.Ports[0].Port)
		logger.Debug("Allocation created", zap.String("connection", conn), zap.String("matchId", match.MatchId))

		if err != nil {
			return err
		}
		req := &pb.AssignTicketsRequest{
			Assignments: []*pb.AssignmentGroup{
				{
					TicketIds: ticketIDs,
					Assignment: &pb.Assignment{
						Connection: conn,
					},
				},
			},
		}

		if _, err := be.AssignTickets(context.Background(), req); err != nil {
			return fmt.Errorf("AssignTickets failed for match %v, got %w", match.GetMatchId(), err)
		}

		notifier.NotifyPlayersOfMatch(match)

		logger.Info("Assigned server %v to match %v", zap.String("conn", conn), zap.Any("match", match))
	}

	return nil
}
