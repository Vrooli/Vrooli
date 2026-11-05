package flow

import "treasury/internal/settlement/flow/generated"

type (
	SettlementStatus = generated.SettlementStatus
	SettlementEvent  = generated.SettlementEvent
)

const (
	SettlementReady        = generated.SettlementReady
	SettlementCalling      = generated.SettlementCalling
	SettlementSettled      = generated.SettlementSettled
	SettlementFailed       = generated.SettlementFailed
	SettlementUnknown      = generated.SettlementUnknown
	SettlementBeginCall    = generated.SettlementBeginCall
	SettlementRailSettled  = generated.SettlementRailSettled
	SettlementRailFailed   = generated.SettlementRailFailed
	SettlementResponseLost = generated.SettlementResponseLost
	SettlementQuerySettled = generated.SettlementQuerySettled
	SettlementQueryFailed  = generated.SettlementQueryFailed
	SettlementQueryUnknown = generated.SettlementQueryUnknown
)

type SettlementState struct{ Status SettlementStatus }

func InitialSettlementState() SettlementState { return SettlementState{Status: SettlementReady} }
