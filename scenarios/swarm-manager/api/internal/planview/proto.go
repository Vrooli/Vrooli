package planview

import (
	"swarm-manager/internal/eta"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// encodeBoard converts the internal Board to its protobuf wire shape.
func encodeBoard(board Board) *apipb.PlanBoardResponse {
	return &apipb.PlanBoardResponse{
		Now:   encodeNow(board.Now),
		Next:  encodeColumn(board.Next),
		Later: encodeColumn(board.Later),
		Done:  encodeColumn(board.Done),
		Meta: &domainpb.PlanBoardMeta{
			GeneratedAt:   board.Meta.GeneratedAt,
			WindowSeconds: int32(board.Meta.WindowSeconds),
			MaxWave:       int32(board.Meta.MaxWave),
			Cycles:        board.Meta.Cycles,
			Eta:           encodeETA(board.Meta.ETA),
		},
	}
}

// encodeETA maps the internal ETA band to its protobuf shape, returning nil
// when no band was computed.
func encodeETA(band *eta.Band) *domainpb.PlanEtaBand {
	if band == nil {
		return nil
	}
	return &domainpb.PlanEtaBand{
		P50Hours:       band.P50Hours,
		P80Hours:       band.P80Hours,
		P50Label:       band.P50Label,
		P80Label:       band.P80Label,
		Basis:          band.Basis,
		BasisLabel:     band.BasisLabel,
		Confidence:     band.Confidence,
		RemainingItems: int32(band.RemainingItems),
		LaneCapacity:   int32(band.LaneCapacity),
	}
}

func encodeNow(now NowSummary) *domainpb.PlanNowSummary {
	lanes := make([]*domainpb.PlanLaneStatus, 0, len(now.Lanes))
	for _, lane := range now.Lanes {
		lanes = append(lanes, &domainpb.PlanLaneStatus{
			Lane:     lane.Lane,
			Active:   int32(lane.Active),
			Capacity: int32(lane.Capacity),
		})
	}
	return &domainpb.PlanNowSummary{
		ActiveCount:   int32(now.ActiveCount),
		QueueDepth:    int32(now.QueueDepth),
		MaxQueueDepth: int32(now.MaxQueueDepth),
		Lanes:         lanes,
	}
}

func encodeColumn(col Column) *domainpb.PlanColumn {
	groups := make([]*domainpb.PlanCardGroup, 0, len(col.Groups))
	for _, group := range col.Groups {
		cards := make([]*domainpb.PlanCard, 0, len(group.Cards))
		for _, card := range group.Cards {
			cards = append(cards, encodeCard(card))
		}
		groups = append(groups, &domainpb.PlanCardGroup{
			Id:          group.ID,
			Label:       group.Label,
			BlockerKind: group.BlockerKind,
			GateId:      group.GateID,
			BlockerKeys: group.BlockerKeys,
			Cards:       cards,
		})
	}
	return &domainpb.PlanColumn{
		Groups:    groups,
		CardCount: int32(col.CardCount),
	}
}

func encodeCard(card Card) *domainpb.PlanCard {
	msg := &domainpb.PlanCard{
		Id:          card.ID,
		CardType:    card.CardType,
		Action:      card.Action,
		ItemKind:    card.ItemKind,
		ItemName:    card.ItemName,
		Title:       card.Title,
		Status:      card.Status,
		Priority:    int32(card.Priority),
		Wave:        int32(card.Wave),
		Milestone:  card.Milestone,
		Effort:      card.Effort,
		Outcome:     card.Outcome,
		FinishedAt:  card.FinishedAt,
		ExecutionId: card.ExecutionID,
		Unblocks:    int32(card.Unblocks),
	}
	if card.Gate != nil {
		msg.Gate = &domainpb.PlanGate{
			Id:             card.Gate.ID,
			Kind:           string(card.Gate.Kind),
			OwnerType:      card.Gate.OwnerType,
			OwnerKind:      card.Gate.OwnerKind,
			OwnerName:      card.Gate.OwnerName,
			OwnerTitle:     card.Gate.OwnerTitle,
			Count:          int32(card.Gate.Count),
			Blocks:         card.Gate.Blocks,
			DecidableSince: card.Gate.DecidableSince,
			Suggested:      card.Gate.Suggested,
		}
	}
	return msg
}
