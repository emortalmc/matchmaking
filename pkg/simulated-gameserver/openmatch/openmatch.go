package openmatch

import (
	"context"
	"fmt"
	"github.com/ztrue/shutdown"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"matchmaker/pkg/simulated-gameserver/agones"
	"open-match.dev/open-match/pkg/pb"
	"time"
)

const (
	openMatchFrontendEndpoint = "open-match-frontend.open-match.svc:50504"
)

var (
	fe        = createFrontend()
	logger, _ = zap.NewDevelopment()

	// BackfillTickers todo: stop tickers when some kind of game start occurs
	BackfillTickers = make(map[string]*chan struct{})
)

func createFrontend() pb.FrontendServiceClient {
	conn, err := grpc.Dial(openMatchFrontendEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)

	if err != nil {
		logger.Error("Failed to connect to Open Match", zap.Error(err))
	}
	shutdown.Add(func() { conn.Close() })
	return pb.NewFrontendServiceClient(conn)
}

func RegisterBackfill(backfillId string) {
	acknowledgeBackfill(backfillId)
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				acknowledgeBackfill(backfillId)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	BackfillTickers[backfillId] = &quit
}

func acknowledgeBackfill(backfillId string) {
	_, err := fe.AcknowledgeBackfill(context.Background(), &pb.AcknowledgeBackfillRequest{
		BackfillId: backfillId,
		Assignment: &pb.Assignment{
			Connection: getConnectionString(),
		},
	})

	if err != nil {
		logger.Error("Failed to acknowledge backfill", zap.Error(err))
	}
}

func getConnectionString() string {
	status := agones.SelfGs.Status
	return fmt.Sprintf("%s:%v", status.Address, status.Ports[0].Port)
}
