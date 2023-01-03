package mmf

import (
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"
	"open-match.dev/open-match/pkg/pb"
)

func getBackfillSlots(backfill *pb.Backfill) (int32, error) {
	if backfill.GetExtensions() != nil {
		if wrappedValue, ok := backfill.GetExtensions()["open_slots"]; ok {
			var val wrappers.Int32Value
			err := wrappedValue.UnmarshalTo(&val)
			if err != nil {
				return 0, err
			}

			return val.Value, nil
		}
	}

	return 0, nil
}

func setBackfillSlots(backfill *pb.Backfill, slots int32) error {
	if backfill.GetExtensions() == nil {
		backfill.Extensions = make(map[string]*anypb.Any)
	}

	wrappedValue, err := anypb.New(&wrappers.Int32Value{Value: slots})
	if err != nil {
		return err
	}

	backfill.Extensions["open_slots"] = wrappedValue

	return nil
}

// taken from https://github.com/googleforgames/open-match/blob/main/examples/functions/golang/backfill/mmf/matchfunction.go
func newSearchFields(pool *pb.Pool) *pb.SearchFields {
	searchFields := pb.SearchFields{}
	rangeFilters := pool.GetDoubleRangeFilters()

	if rangeFilters != nil {
		doubleArgs := make(map[string]float64)
		for _, f := range rangeFilters {
			doubleArgs[f.DoubleArg] = (f.Max - f.Min) / 2
		}

		if len(doubleArgs) > 0 {
			searchFields.DoubleArgs = doubleArgs
		}
	}

	stringFilters := pool.GetStringEqualsFilters()

	if stringFilters != nil {
		stringArgs := make(map[string]string)
		for _, f := range stringFilters {
			stringArgs[f.StringArg] = f.Value
		}

		if len(stringArgs) > 0 {
			searchFields.StringArgs = stringArgs
		}
	}

	tagFilters := pool.GetTagPresentFilters()

	if tagFilters != nil {
		tags := make([]string, len(tagFilters))
		for _, f := range tagFilters {
			tags = append(tags, f.Tag)
		}

		if len(tags) > 0 {
			searchFields.Tags = tags
		}
	}

	return &searchFields
}
