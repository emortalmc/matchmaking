package main

import (
	"context"
	"flag"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"log"
	"open-match.dev/open-match/pkg/pb"
	"time"
)

var (
	// retrieve the TICKETS_PER_SECOND environment variable
	timeBetweenCreations = flag.Duration("time_between_creations", 1*time.Second, "The time between ticket creations")
	ticketCreationAmount = flag.Int("ticket_creation_amount", 1, "The amount of tickets to create per duration")
)

const (
	openMatchFrontendEndpoint = "open-match-frontend.open-match.svc:50504"
)

var (
	feClient pb.FrontendServiceClient

	ticketCounter = 0
	modes         = []string{"game.lobby", "game.marathon"}
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(openMatchFrontendEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)

	if err != nil {
		log.Fatalf("Failed to connect to Open Match, got %v", err)
	}

	defer conn.Close()
	feClient = pb.NewFrontendServiceClient(conn)

	log.Printf("Creating %v ticket(s) every %v", *ticketCreationAmount, *timeBetweenCreations)
	log.Printf("Using modes: %v", modes)

	for ; true; <-time.Tick(*timeBetweenCreations) {
		for i := 0; i < *ticketCreationAmount; i++ {
			ticket := getTicket()
			req := &pb.CreateTicketRequest{Ticket: ticket}
			resp, err := feClient.CreateTicket(context.Background(), req)

			if err != nil {
				log.Printf("Failed to create ticket (%s), got %v", ticket, err)
				continue
			}
			log.Printf("Created ticket %s", resp.Id)
		}
	}
}

func getTicket() *pb.Ticket {
	mode := modes[ticketCounter]

	// increment after so we're always starting at index 0, not 1 to avoid weirdness.
	ticketCounter++
	if ticketCounter >= len(modes) {
		ticketCounter = 0
	}

	playerId, err := anypb.New(wrapperspb.String(uuid.New().String()))
	if err != nil {
		log.Fatalf("Failed to create playerId, got %v", err)
	}

	return &pb.Ticket{
		SearchFields: &pb.SearchFields{
			Tags: []string{mode},
		},
		PersistentField: map[string]*anypb.Any{
			"playerId": playerId,
		},
	}
}
