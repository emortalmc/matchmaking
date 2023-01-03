package utils

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"
	"open-match.dev/open-match/pkg/pb"
)

var logger, _ = zap.NewProduction()

func ExtractPlayerIdFromTicket(ticket *pb.Ticket) (string, error) {
	a := ticket.PersistentField["playerId"]
	var value wrappers.StringValue
	err := proto.Unmarshal(a.Value, &value)
	if err != nil {
		return "", err
	}
	return value.Value, nil
}

func ExtractPlayerIdsFromTickets(tickets []*pb.Ticket) []string {
	var playerIds []string
	for _, ticket := range tickets {
		pId, err := ExtractPlayerIdFromTicket(ticket)
		if err != nil {
			logger.Error("Failed to extract player id from ticket", zap.Any("ticket", ticket), zap.Error(err))
			continue
		}
		playerIds = append(playerIds, pId)
	}
	return playerIds
}
