package selector

import (
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocatorv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"log"
	"matchmaker/pkg/common/modeprofile"
	"math"
	"open-match.dev/open-match/pkg/pb"
)

var (
	AllocatedState = agonesv1.GameServerStateAllocated
	ReadyState     = agonesv1.GameServerStateReady
)

// contains some common selectors

func CommonSelector(profile modeprofile.ModeProfile, match *pb.Match) *allocatorv1.GameServerAllocation {
	return &allocatorv1.GameServerAllocation{
		Spec: allocatorv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocatorv1.GameServerSelector{
				{
					LabelSelector: v1.LabelSelector{
						MatchLabels: map[string]string{
							"agones.dev/fleet":               profile.FleetName,
							"agones.dev/sdk-should-allocate": "true",
						},
					},
					GameServerState: &AllocatedState,
				},
				{
					LabelSelector: v1.LabelSelector{
						MatchLabels: map[string]string{"agones.dev/fleet": profile.FleetName},
					},
					GameServerState: &ReadyState,
				},
			},
			MetaPatch: allocatorv1.MetaPatch{
				Annotations: createPatchedAnnotations(match),
			},
		},
	}
}

// CommonPlayerBasedSelector selects a GameServer where there is no 'match'.
// This could be a singleplayer game (e.g. marathon) or a stateless drop-in drop-out game (e.g. the lobby)
func CommonPlayerBasedSelector(profile modeprofile.ModeProfile, match *pb.Match, playerCount int64) *allocatorv1.GameServerAllocation {
	return &allocatorv1.GameServerAllocation{
		Spec: allocatorv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocatorv1.GameServerSelector{
				{
					LabelSelector: v1.LabelSelector{
						MatchLabels: map[string]string{
							"agones.dev/fleet": profile.FleetName,
						},
					},
					Players: &allocatorv1.PlayerSelector{
						MinAvailable: playerCount, // will need to change for party support
						MaxAvailable: math.MaxInt,
					},
					GameServerState: &AllocatedState,
				},
				{
					LabelSelector: v1.LabelSelector{
						MatchLabels: map[string]string{"agones.dev/fleet": profile.FleetName},
					},
					GameServerState: &ReadyState,
				},
			},
			MetaPatch: allocatorv1.MetaPatch{
				Annotations: createPatchedAnnotations(match),
			},
		},
	}
}

func createPatchedAnnotations(match *pb.Match) map[string]string {
	expectedPlayers, err := createExpectedPlayers(match)
	if err != nil {
		log.Printf("Error creating expected players: %v", err)
		return nil
	}

	var matchId string
	//if backfill != nil {
	//	matchIdAny := backfill.Extensions["originalMatchId"]
	//
	//	matchIdProto, err := anypb.UnmarshalNew(matchIdAny, proto2.UnmarshalOptions{})
	//	if err != nil {
	//		log.Printf("Error unmarshalling backfill id: %v", err)
	//		return nil
	//	}
	//	matchId = matchIdProto.(*wrappers.StringValue).Value
	//} else {
	matchId = uuid.New().String()
	//}

	return map[string]string{
		"openmatch.dev/match-id":         matchId,
		"openmatch.dev/expected-players": expectedPlayers,
		//"openmatch.dev/backfill-id":      backfill.GetId(),
	}
}

func createExpectedPlayers(match *pb.Match) (string, error) {
	var expectedIds []string
	for _, ticket := range match.GetTickets() {
		a := ticket.PersistentField["playerId"]
		var value wrappers.StringValue
		err := proto.Unmarshal(a.Value, &value)
		if err != nil {
			return "", err
		}
		expectedIds = append(expectedIds, value.Value)
	}
	jBytes, err := json.Marshal(expectedIds)
	if err != nil {
		return "", err
	}
	return string(jBytes), nil
}
