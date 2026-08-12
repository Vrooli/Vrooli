//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"backdrop-studio/integration"

	"github.com/stretchr/testify/require"
)

// writeEvidenceEnv opts into writing files. Without it this test still renders
// everything and asserts on the result — it simply does not touch the
// repository, so an ordinary lane run never produces an uncommitted diff.
const writeEvidenceEnv = "BACKDROP_STUDIO_WRITE_EVIDENCE"

// TestRenderMatrixEvidence is the producer of `docs/evidence/render-matrix.md`.
//
// It exists because unreproducible evidence is itself the defect. Twelve style
// previews rendered during an earlier catalog seeding came from a throwaway
// probe and had to be deleted rather than shipped, because nobody could say
// which command made them. Every artifact this scenario commits is produced by
// a command named in `docs/internal/EVIDENCE.md`, and this is that command for
// the render matrix.
//
//	make integration-evidence
func TestRenderMatrixEvidence(t *testing.T) {
	env, caps := newEnvironment(t)
	ctx := context.Background()

	report, err := env.AssertFreshBinary(ctx)
	require.NoError(t, err)

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	sort.Slice(styles, func(i, j int) bool { return styles[i].ID < styles[j].ID })

	type row struct {
		style    integration.Style
		ok       bool
		width    int
		height   int
		detail   string
		duration time.Duration
	}
	rows := make([]row, 0, len(styles))
	failures := 0
	skips := 0

	for _, style := range styles {
		entry := row{style: style}
		if style.ModelBacked() && !caps.GenerationOK {
			entry.detail = "SKIP(no-image-model): image-tools reports no enabled, installed model serving text_to_image or image_to_image"
			skips++
			rows = append(rows, entry)
			continue
		}
		started := time.Now()
		job, submitErr := env.Submit(ctx, integration.SubmitOptions{StyleID: style.ID, Seed: laneSeed})
		entry.duration = time.Since(started)
		if submitErr != nil {
			if integration.IsGPUCapacityFailure(submitErr) {
				entry.detail = "SKIP(gpu-capacity): reached the model, but the host could not allocate device memory"
				skips++
			} else {
				entry.detail = firstLine(submitErr.Error())
				failures++
			}
			rows = append(rows, entry)
			continue
		}
		require.NotEmpty(t, job.Candidates)
		width, height, decodeErr := integration.DecodePNG(job.Candidates[0].ImagePNG)
		if decodeErr != nil {
			entry.detail = firstLine(decodeErr.Error())
			failures++
			rows = append(rows, entry)
			continue
		}
		entry.ok, entry.width, entry.height = true, width, height
		entry.detail = job.ExecutionPath
		rows = append(rows, entry)
	}

	var doc strings.Builder
	fmt.Fprintf(&doc, "# Render matrix\n\n")
	fmt.Fprintf(&doc, "Every seeded style rendered through a really running `image-tools`, at seed %d,\n", laneSeed)
	fmt.Fprintf(&doc, "with no brand bound — the path a CLI caller always takes.\n\n")
	fmt.Fprintf(&doc, "**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.\n\n")
	fmt.Fprintf(&doc, "| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&doc, "| API build | `%s` |\n", report.Fingerprint[:12])
	fmt.Fprintf(&doc, "| Catalog seed version | %d applied of %d shipped |\n", report.AppliedSeedVersion, report.SeedVersion)
	fmt.Fprintf(&doc, "| Image models installed | %s |\n", orNone(caps.ImageModels))
	fmt.Fprintf(&doc, "| Conditioning adapters ready | %s |\n", orNone(caps.Adapters))
	fmt.Fprintf(&doc, "| Result | **%d rendered, %d failed, %d skipped** of %d |\n\n",
		len(rows)-failures-skips, failures, skips, len(rows))

	fmt.Fprintf(&doc, "| Style | Strategy | Result | Geometry | Detail |\n|---|---|---|---|---|\n")
	for _, entry := range rows {
		result := "FAIL"
		geometry := "—"
		switch {
		case entry.ok:
			result = "pass"
			geometry = fmt.Sprintf("%dx%d", entry.width, entry.height)
		case strings.HasPrefix(entry.detail, "SKIP"):
			result = "skip"
		}
		fmt.Fprintf(&doc, "| `%s` | %s | %s | %s | %s |\n",
			entry.style.ID, entry.style.Strategy, result, geometry, escapePipes(entry.detail))
	}

	if skips > 0 {
		fmt.Fprintf(&doc, "\nEvery skip names the capability it is waiting on. A skip is not a pass.\n")
	}

	if strings.TrimSpace(os.Getenv(writeEvidenceEnv)) != "" {
		path, absErr := filepath.Abs("../../docs/evidence/render-matrix.md")
		require.NoError(t, absErr)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(doc.String()), 0o644))
		t.Logf("wrote %s", path)
	} else {
		t.Logf("set %s=1 to commit this matrix\n%s", writeEvidenceEnv, doc.String())
	}

	require.Zerof(t, failures, "%d seeded styles did not render; the matrix above names each one", failures)
}

func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return strings.TrimSpace(text[:idx])
	}
	return strings.TrimSpace(text)
}

func escapePipes(text string) string { return strings.ReplaceAll(text, "|", "\\|") }

func orNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return "`" + strings.Join(values, "`, `") + "`"
}
