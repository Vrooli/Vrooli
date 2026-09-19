// Package designinference owns durable, bounded design-inference dispatch.
// ai-gateway owns model routing and provider execution.
package designinference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrConflict = errors.New("inference idempotency key is already bound to different inputs")

type Scope struct {
	Scenario string `json:"scenario"`
	Page     string `json:"page"`
}
type Request struct {
	// Optional only for historical records; new design dispatches name their scope.
	Scope           *Scope `json:"scope,omitempty"`
	Role            string `json:"role"`
	MaxOutputTokens int32  `json:"maxOutputTokens"`
	Source          string `json:"source"`
	Schema          string `json:"schema"`
	Instruction     string `json:"instruction"`
	// Context pins caller-owned proposal inputs for reconstruction after restart.
	Context json.RawMessage `json:"context"`
}
type Operation struct {
	ID           string
	RequestHash  string
	Request      Request
	State        string
	ResponseJSON string
	Detail       string
}
type Repository interface {
	Create(context.Context, string, Request) (Operation, error)
	Get(context.Context, string) (Operation, error)
	Claim(context.Context, string) (bool, error)
	Finish(context.Context, string, string, string, string) (Operation, error)
}

func identity(r Request) (string, []byte, error) {
	if r.Scope != nil {
		for _, value := range []string{r.Scope.Scenario, r.Scope.Page} {
			if strings.TrimSpace(value) == "" || len(value) > 128 {
				return "", nil, fmt.Errorf("inference scope requires bounded scenario and page identities")
			}
		}
	}
	if r.Role != "extract.structured" || r.MaxOutputTokens != 4096 || strings.TrimSpace(r.Source) == "" || len(r.Source) > 512*1024 || len(r.Schema) == 0 || len(r.Schema) > 64*1024 || len(r.Instruction) == 0 || len(r.Instruction) > 8000 || len(r.Context) > 2*1024*1024 || !json.Valid(r.Context) || !json.Valid([]byte(r.Schema)) {
		return "", nil, fmt.Errorf("inference requires bounded source, schema, instruction and proposal context")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), raw, nil
}
