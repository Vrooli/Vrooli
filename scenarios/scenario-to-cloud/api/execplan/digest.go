package execplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// DigestPrefix marks a semantic digest on the wire.
const DigestPrefix = "sha256:"

// CanonicalJSON returns the canonical bytes the digest covers: the plan with
// the presentation block zeroed, encoded by encoding/json (object keys are
// emitted in struct order, map keys sorted, no insignificant whitespace).
func CanonicalJSON(plan *Plan) ([]byte, error) {
	if plan == nil {
		return nil, fmt.Errorf("execplan: nil plan")
	}
	semantic := *plan
	semantic.Presentation = Presentation{}
	semantic.Preconditions = nonNilPreconditions(semantic.Preconditions)
	semantic.Actions = nonNilActions(semantic.Actions)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(semantic); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

// SemanticDigest is sha256 over CanonicalJSON, prefixed with "sha256:".
func (p *Plan) SemanticDigest() (string, error) {
	canonical, err := CanonicalJSON(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return DigestPrefix + hex.EncodeToString(sum[:]), nil
}

// MustDigest is SemanticDigest for callers that already validated the plan.
func (p *Plan) MustDigest() string {
	digest, err := p.SemanticDigest()
	if err != nil {
		panic(err)
	}
	return digest
}

func nonNilPreconditions(in []Precondition) []Precondition {
	if in == nil {
		return []Precondition{}
	}
	return in
}

func nonNilActions(in []Action) []Action {
	if in == nil {
		return []Action{}
	}
	out := make([]Action, len(in))
	for i, action := range in {
		if action.DependsOn == nil {
			action.DependsOn = []string{}
		}
		if action.Inputs == nil {
			action.Inputs = map[string]string{}
		}
		out[i] = action
	}
	return out
}
