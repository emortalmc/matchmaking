package notifier

import (
	"context"
	"fmt"
	"github.com/EmortalMC/grpc-api-specs/gen/go/service/gameserver/matchmaking"
	"github.com/EmortalMC/grpc-api-specs/gen/go/service/player_tracker"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"matchmaker/pkg/common/utils/kubernetes"
	"open-match.dev/open-match/pkg/pb"
	"os"
	"time"
)

var (
	enabled = true

	logger, _           = zap.NewProduction()
	playerTrackerClient = createPlayerTrackerClient()
	namespace           = os.Getenv("NAMESPACE")
)

// NotifyPlayersOfMatch notifies the player of a match that will begin immediately
func NotifyPlayersOfMatch(match *pb.Match) {
	NotifyPlayersOfPendingMatch(getPlayerIdsFromMatch(match), time.Now())
}

// NotifyPlayersOfPendingMatch notifies the player of a match that will begin at the teleportTime
// A Match does not exist at this time so the raw playerIds are passed in
func NotifyPlayersOfPendingMatch(playerIds []string, teleportTime time.Time) {
	if !enabled {
		return
	}
	serverResp, err := playerTrackerClient.GetPlayerServers(context.Background(), &player_tracker.PlayersRequest{PlayerIds: playerIds})

	if err != nil {
		logger.Error("Failed to get player servers", zap.Error(err))
		return
	}

	for playerId, server := range serverResp.GetPlayerServers() {
		go notify(playerId, server, uint32(len(playerIds)), teleportTime)
	}
}

// NotifyPlayersOfCancelledCountdown notifies the player that the countdown has been cancelled
func NotifyPlayersOfCancelledCountdown(playerIds []string) {
	if !enabled {
		return
	}
	serverResp, err := playerTrackerClient.GetPlayerServers(context.Background(), &player_tracker.PlayersRequest{PlayerIds: playerIds})

	if err != nil {
		logger.Error("Failed to get player servers", zap.Error(err))
		return
	}

	for playerId, server := range serverResp.GetPlayerServers() {
		go notifyCancelledCountdown(playerId, server)
	}
}

func notify(playerId string, server *player_tracker.OnlineServer, playerCount uint32, teleportTime time.Time) {
	client, err := getMatchmakingClient(server)
	if err != nil {
		logger.Error("Failed to get matchmaking client", zap.Error(err))
		return
	}

	_, err = client.MatchFound(context.Background(), &matchmaking.MatchFoundRequest{
		PlayerId:     playerId,
		PlayerCount:  playerCount,
		TeleportTime: timestamppb.New(teleportTime),
	})
	if err != nil {
		logger.Error("Failed to notify matchmaking client", zap.Error(err))
		return
	}
}

func notifyCancelledCountdown(playerId string, server *player_tracker.OnlineServer) {
	client, err := getMatchmakingClient(server)
	if err != nil {
		logger.Error("Failed to get matchmaking client", zap.Error(err))
		return
	}

	_, err = client.MatchCancelled(context.Background(), &matchmaking.MatchCancelledRequest{PlayerId: playerId})

	if err != nil {
		logger.Error("Failed to notify matchmaking client", zap.Error(err))
		return
	}
}

func getMatchmakingClient(server *player_tracker.OnlineServer) (matchmaking.GameServerMatchmakingClient, error) {
	result, err := kubernetes.KubeClient.CoreV1().Pods(namespace).Get(context.Background(), server.ServerId, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ip := result.Status.PodIP
	port, err := getGrpcPort(result)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", ip, port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return matchmaking.NewGameServerMatchmakingClient(conn), nil
}

func getGrpcPort(pod *v12.Pod) (int32, error) {
	container := pod.Spec.Containers[0]
	for _, port := range container.Ports {
		if port.Name == "grpc" {
			return port.ContainerPort, nil
		}
	}
	return 0, fmt.Errorf("no grpc port found for server %s", pod.Name)
}

func getPlayerIdsFromMatch(match *pb.Match) []string {
	playerIds := make([]string, len(match.GetTickets()))

	for i, ticket := range match.GetTickets() {
		a := ticket.PersistentField["playerId"]
		var value wrappers.StringValue
		err := proto.Unmarshal(a.Value, &value)
		if err != nil {
			logger.Error("Failed to unmarshal playerId", zap.Error(err))
			continue
		}
		playerIds[i] = value.Value
	}

	return playerIds
}

func createPlayerTrackerClient() player_tracker.PlayerTrackerClient {
	conn, err := grpc.Dial("localhost:50502", grpc.WithInsecure(), grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	if err != nil {
		logger.Error("Failed to connect to Player Tracker", zap.Error(err))
		enabled = false
	}

	return player_tracker.NewPlayerTrackerClient(conn)
}
