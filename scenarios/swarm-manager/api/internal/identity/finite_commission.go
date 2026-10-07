// finite_commission.go binds human commission disposition to the governed effort.
package identity

import (
	"github.com/vrooli/api-core/effortauthority"
	"time"
)

// FiniteCommissionRecord is nested in the existing versioned owner aggregate;
// it is neither completion acceptance nor a second writable approval database.
type FiniteCommissionRecord struct {
	// Optional exact reviewed Development contract binding. Legacy records
	// remain valid for their original finite owner, not new Development decisions.
	DevelopmentContractDigest string                            `json:"development_contract_digest,omitempty"`
	WorkShape                 string                            `json:"work_shape"`
	Subject                   effortauthority.CommissionSubject `json:"subject"`
	AuthorityDigest           string                            `json:"authority_digest"`
	PlanSubjectVersion        string                            `json:"plan_subject_version"`
	PlanContentHash           string                            `json:"plan_content_hash"`
	Actor                     string                            `json:"actor"`
	AcceptedAt                time.Time                         `json:"accepted_at"`
	Generation                uint64                            `json:"generation"`
	Revoked                   bool                              `json:"revoked"`
	RevokedAt                 time.Time                         `json:"revoked_at,omitempty"`
}
