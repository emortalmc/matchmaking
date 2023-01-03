package matchprofile

import (
	"fmt"
	"open-match.dev/open-match/pkg/pb"
)

func CommonProfile(name string, gameName string) *pb.MatchProfile {
	return &pb.MatchProfile{
		Name: name,
		Pools: []*pb.Pool{
			{
				Name: "all",
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: fmt.Sprintf("game.%s", gameName),
					},
				},
			},
		},
	}
}
