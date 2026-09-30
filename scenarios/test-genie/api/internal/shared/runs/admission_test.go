package runs

import (
	"testing"
	"time"
)

func TestAdmissionIdentityIsImmutableAcrossIndexWrites(t *testing.T) {
	for _, method := range []string{"append", "update", "finalize"} {
		t.Run(method, func(t *testing.T) {
			index := NewIndex(t.TempDir())
			original := RunRecord{RunID: "retained", Scenario: "demo", Status: StatusInProgress, StartedAt: time.Now().UTC(), AdmissionIntentDigest: "sha256:original"}
			if _, created, err := index.AdmitExplicit(original); err != nil || !created {
				t.Fatalf("admit: %v %t", err, created)
			}
			mutate := func(r *RunRecord) error {
				r.AdmissionIntentDigest = "sha256:changed"
				r.Status = StatusPassed
				return nil
			}
			var err error
			switch method {
			case "append":
				changed := original
				_ = mutate(&changed)
				err = index.Append(changed)
			case "update":
				err = index.Update(original.RunID, mutate)
			case "finalize":
				err = index.Finalize(original.RunID, map[string]bool{"success": true}, mutate)
			}
			if err == nil {
				t.Fatal("immutable admission identity was overwritten")
			}
			retained, err := index.Find(original.RunID)
			if err != nil || retained.AdmissionIntentDigest != original.AdmissionIntentDigest || retained.Status != original.Status {
				t.Fatalf("failed mutation changed original: %+v %v", retained, err)
			}
		})
	}
}
