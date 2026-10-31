// Package workflowcontract defines the transport-neutral workflow boundary
// shared by subject domains and the declared transition runner.
//
// Subject domains may depend on these contracts, but must not import the
// Agent Manager client. The transition runner owns provider conversion and
// workflow lifecycle operations.
package workflowcontract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

var ErrStartNotFound = errors.New("workflow start not found")

type Invocation struct {
	Owner       string
	WorkflowKey string
	// ApprovalDigest identifies the reviewed Swarm engagement. It is distinct
	// from the Agent Manager workflow revision selected by WorkflowDigest.
	ApprovalDigest string
	WorkflowDigest string
	GrantDigest    string
	Input          *structpb.Value
	IdempotencyKey string
	FirstRunNodeID string
	Activity       *Activity
	Grant          *Grant
}

type Activity struct {
	OwnerType  string
	OwnerKind  string
	OwnerName  string
	OwnerTitle string
	Purpose    string
}

type Grant struct {
	MaxTurns           int
	MaxTokens          int64
	MaxChargeMicroUSD  int64
	MaxWallTimeSeconds int64
	MaxNodeAttempts    int
	MaxChildren        int
	MaxConcurrency     int
	MaxRecursion       int
	MaxRetries         int
	RetryLimitSet      bool
	MaxWaitSeconds     int
	// AllowedEffects is the immutable, owner-enforced effect ceiling. It is
	// carried with the grant so omitted caller fields cannot become unlimited.
	AllowedEffects []string
}

// GrantDigest identifies the exact neutral grant values sent to the owner.
// It is separate from both the reviewed approval and workflow definition.
func GrantDigest(grant *Grant) string {
	if grant == nil {
		return ""
	}
	effects := append([]string(nil), grant.AllowedEffects...)
	sort.Strings(effects)
	identity := fmt.Sprintf("turns=%d\ntokens=%d\ncharge_micro_usd=%d\nwall_seconds=%d\nnode_attempts=%d\nchildren=%d\nconcurrency=%d\nrecursion=%d\nretries=%d\nwait_seconds=%d\neffects=%s\n", grant.MaxTurns, grant.MaxTokens, grant.MaxChargeMicroUSD, grant.MaxWallTimeSeconds, grant.MaxNodeAttempts, grant.MaxChildren, grant.MaxConcurrency, grant.MaxRecursion, grant.MaxRetries, grant.MaxWaitSeconds, strings.Join(effects, "\n"))
	if grant.RetryLimitSet && grant.MaxRetries == 0 {
		identity += "retry_limit_explicit_zero=true\n"
	}
	sum := sha256.Sum256([]byte(identity))
	return "sha256:" + hex.EncodeToString(sum[:])
}

type Start struct {
	ExecutionID    string
	RunID          string
	WorkflowDigest string
	ApprovalDigest string
	GrantDigest    string
	// DefinitionDigest is retained for older adapters; new owners should use
	// WorkflowDigest because it identifies executable workflow structure.
	DefinitionDigest   string
	CapabilityRevision string
	SelectionReason    string
	Status             string
}

type Usage struct {
	Turns          int64
	Children       int64
	NodeAttempts   int64
	Retries        int64
	Slices         int64
	Tokens         int64
	WallSeconds    int64
	ChargeMicroUSD int64
	TokensKnown    bool
	ChargeMeasured bool
}

type Completion struct {
	ExecutionID      string
	WorkflowDigest   string
	ApprovalDigest   string
	GrantDigest      string
	DefinitionDigest string
	Status           string
	TerminalCode     string
	BudgetName       string
	Input            *structpb.Value
	Output           *structpb.Value
	Usage            *Usage
}

type Invoker interface {
	Start(context.Context, Invocation) (Start, error)
	Collect(context.Context, string) (Completion, error)
}

// StartReconciler resolves a start accepted by the owner when the original
// transport response was lost. The key is the same durable idempotency key
// used for Start; implementations must not create a new execution.
type StartReconciler interface {
	ReconcileStart(context.Context, string) (Start, error)
}

type Canceller interface {
	Cancel(context.Context, string, string, string) error
}
