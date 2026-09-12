package rewrite

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// DerivePlanID returns the canonical PlanID for an already-normalized
// operation list. Identical inputs always produce identical IDs.
//
// Hash inputs deliberately exclude project_path and any timestamp —
// the PlanStore scopes plans by (project_path, PlanID) so two
// projects with the same operation list cannot replay each other's
// plans even though they share an ID.
func DerivePlanID(ops []Operation) PlanID {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(canonicalOps(ops))
	sum := sha256.Sum256(buf.Bytes())
	return PlanID(hex.EncodeToString(sum[:]))
}

// canonicalOp is the on-hash shape of an Operation. Both arms are
// emitted in a stable order with stable field names so a future field
// addition is an explicit JSON-shape change (and therefore a planned
// PlanID rotation), not a silent drift.
type canonicalOp struct {
	Tag           string         `json:"tag"`
	FileMove      *canonicalMove `json:"file_move,omitempty"`
	ImportRewrite *canonicalImp  `json:"import_rewrite,omitempty"`
}

type canonicalMove struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
}

type canonicalImp struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

func canonicalOps(ops []Operation) []canonicalOp {
	out := make([]canonicalOp, 0, len(ops))
	for _, op := range ops {
		c := canonicalOp{Tag: op.OperationTag()}
		if op.FileMove != nil {
			c.FileMove = &canonicalMove{FromPath: op.FileMove.FromPath, ToPath: op.FileMove.ToPath}
		}
		if op.ImportRewrite != nil {
			c.ImportRewrite = &canonicalImp{OldPath: op.ImportRewrite.OldPath, NewPath: op.ImportRewrite.NewPath}
		}
		out = append(out, c)
	}
	return out
}
