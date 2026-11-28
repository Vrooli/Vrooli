package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeclaredDeviceMatrixEnumeratesEveryTargetAsUnqualifiedAndUnrun(t *testing.T) {
	path := filepath.Join("..", "..", "..", "coverage", "device-qualification-matrix.json")
	declaration, err := loadDeviceMatrix(path)
	require.NoError(t, err)
	cells := enumerateDeviceMatrix(declaration)
	require.Len(t, declaration.Profiles, 9)
	require.Len(t, declaration.Targets, 3)
	require.Len(t, cells, 27)

	seen := map[string]bool{}
	for _, cell := range cells {
		seen[cell.ID] = true
		require.Equal(t, "not_run", cell.ExecutionState)
		require.Equal(t, "unverified", cell.SupportState)
		require.Equal(t, "unqualified", cell.Qualification)
		require.False(t, cell.Qualified)
		require.Contains(t, cell.MissingEvidence, "native device/browser product-path receipt")
	}
	var kaleCell *deviceMatrixCell
	for i := range cells {
		if cells[i].Target.ID == "kale-nano-tts" {
			kaleCell = &cells[i]
			break
		}
	}
	require.NotNil(t, kaleCell)
	require.Contains(t, kaleCell.MissingEvidence, "runtime availability decision")
	for _, id := range []string{
		"D1-linux-chromium-cpu/whisper-small-batch-stt",
		"D2-linux-chromium-constrained/whisper-small-batch-stt",
		"D6-android-chrome/whisper-small-batch-stt",
		"D1-linux-chromium-cpu/sherpa-streaming-zipformer",
		"D2-linux-chromium-constrained/sherpa-streaming-zipformer",
		"D3-macos-safari/sherpa-streaming-zipformer",
		"D3-macos-chromium/sherpa-streaming-zipformer",
		"D4-windows-chromium/sherpa-streaming-zipformer",
		"D5-ios-safari/sherpa-streaming-zipformer",
		"D5-ios-installed-pwa/sherpa-streaming-zipformer",
		"D6-android-chrome/sherpa-streaming-zipformer",
		"D1-linux-chromium-cpu/kale-nano-tts",
		"D6-android-chrome/kale-nano-tts",
	} {
		require.True(t, seen[id], "required matrix cell %s missing", id)
	}
	var kale *matrixTarget
	for i := range declaration.Targets {
		if declaration.Targets[i].ID == "kale-nano-tts" {
			kale = &declaration.Targets[i]
		}
	}
	require.NotNil(t, kale)
	require.Equal(t, "conditional_candidate_when_runtime_available", kale.QualificationPolicy)
}

func TestDeviceMatrixDeclarationRejectsQualificationBoundaryClaims(t *testing.T) {
	path := filepath.Join(t.TempDir(), "matrix.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":1,"qualification_boundary":"qualified","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed"}],"profiles":[{"id":"D1","profile":"D1","os":"Linux","browser":"Chromium"}]}`), 0o600))
	_, err := loadDeviceMatrix(path)
	require.ErrorContains(t, err, "inventory_only")
}

func TestDeviceMatrixDeclarationRejectsDuplicateCellIdentityParts(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "target id",
			json: `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed"},{"id":"x","capability":"tts","provider":"q","model_id":"n","qualification_policy":"required_when_claimed"}],"profiles":[{"id":"D1","profile":"D1","os":"Linux","browser":"Chromium"}]}`,
			want: `duplicate target id "x"`,
		},
		{
			name: "profile id",
			json: `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed"}],"profiles":[{"id":"D1","profile":"D1 Linux","os":"Linux","browser":"Chromium"},{"id":"D1","profile":"D1 Android","os":"Android","browser":"Chrome"}]}`,
			want: `duplicate device profile id "D1"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "matrix.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.json), 0o600))
			_, err := loadDeviceMatrix(path)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestDeviceMatrixDeclarationRejectsMissingAndIncompleteInventory(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"missing targets", `{"schema_version":1,"qualification_boundary":"inventory_only","profiles":[{"id":"D1","profile":"D1","os":"Linux","browser":"Chromium"}]}`, "declare targets and profiles"},
		{"missing profiles", `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed"}]}`, "declare targets and profiles"},
		{"incomplete target", `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m"}],"profiles":[{"id":"D1","profile":"D1","os":"Linux","browser":"Chromium"}]}`, "incomplete target"},
		{"incomplete profile", `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed"}],"profiles":[{"id":"D1","profile":"D1","browser":"Chromium"}]}`, "incomplete device profile"},
		{"undeclared status field", `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed","qualified":true}],"profiles":[{"id":"D1","profile":"D1","os":"Linux","browser":"Chromium"}]}`, "unknown field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "matrix.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.json), 0o600))
			_, err := loadDeviceMatrix(path)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestDeviceMatrixDeclarationRejectsUnknownQualificationPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "matrix.json")
	payload := `{"schema_version":1,"qualification_boundary":"inventory_only","targets":[{"id":"x","capability":"stt","provider":"p","model_id":"m","qualification_policy":"required_when_claimed_typo"}],"profiles":[{"id":"D1","profile":"D1","os":"Linux","browser":"Chromium"}]}`
	require.NoError(t, os.WriteFile(path, []byte(payload), 0o600))
	_, err := loadDeviceMatrix(path)
	require.ErrorContains(t, err, "unsupported qualification policy")
}

func TestDeviceMatrixEnumerationHasStableProfileThenTargetOrdering(t *testing.T) {
	declaration := deviceMatrixDeclaration{
		Targets:  []matrixTarget{{ID: "stt"}, {ID: "tts"}},
		Profiles: []matrixProfile{{ID: "D2"}, {ID: "D1"}},
	}
	cells := enumerateDeviceMatrix(declaration)
	ids := make([]string, 0, len(cells))
	for _, cell := range cells {
		ids = append(ids, cell.ID)
	}
	require.Equal(t, []string{"D2/stt", "D2/tts", "D1/stt", "D1/tts"}, ids)
}

func TestEvaluateDeviceMatrixRequiresPassingProductReceiptForEveryDeclaredCell(t *testing.T) {
	declaration := deviceMatrixDeclaration{
		SchemaVersion: 1, Boundary: "product_path_receipts",
		Targets:  []matrixTarget{{ID: "batch", Capability: "stt.batch", Provider: "whisper-stt", ModelID: "whisper-small", QualificationPolicy: matrixPolicyRequiredWhenClaimed}},
		Profiles: []matrixProfile{{ID: "D1-linux", Profile: "D1", OS: "Linux amd64", Browser: "Chromium", Accelerator: "CPU only"}, {ID: "D3-macos", Profile: "D3", OS: "macOS arm64", Browser: "Safari", Accelerator: "CPU only"}},
	}
	valid := `{"schema_version":1,"receipts":[{"cell_id":"D1-linux/batch","outcome":"passed","observed_at":"2026-11-23T12:00:00Z","os_version":"Linux 6.8.0","architecture":"amd64","cpu_count":8,"memory_total_bytes":17179869184,"accelerator_identity":"none","browser":"Chromium","browser_version":"150.0.1","device_identity":"host-1","provider":"whisper-stt","model_id":"whisper-small","engine_version":"1.0","model_revision":"sha256:model","product_path":"browser microphone to batch transcription API","product_path_status":"passed","artifact_ref":"sha256:artifact","artifact_sha256":"` + strings.Repeat("a", 64) + `"},{"cell_id":"D3-macos/batch","outcome":"unavailable","reason":"No named macOS arm64 Safari device/operator is available"}]}`
	cells, err := evaluateDeviceMatrix(declaration, []byte(valid))
	require.NoError(t, err)
	require.Len(t, cells, 2)
	require.True(t, cells[0].Qualified)
	require.Equal(t, "qualified", cells[0].Qualification)
	require.False(t, cells[1].Qualified)
	require.Contains(t, cells[1].MissingEvidence, "No named macOS")

	missing, err := evaluateDeviceMatrix(declaration, []byte(`{"schema_version":1,"receipts":[]}`))
	require.NoError(t, err)
	require.Equal(t, "not_run", missing[0].ExecutionState)
	require.False(t, missing[0].Qualified)
}

func TestEvaluateDeviceMatrixRejectsUnsupportedOrUnprovenPasses(t *testing.T) {
	declaration := deviceMatrixDeclaration{
		SchemaVersion: 1, Boundary: "product_path_receipts",
		Targets:  []matrixTarget{{ID: "batch", Provider: "whisper-stt", ModelID: "whisper-small"}},
		Profiles: []matrixProfile{{ID: "D1-linux", OS: "Linux amd64", Browser: "Chromium", Accelerator: "CPU only"}},
	}
	base := `"cell_id":"D1-linux/batch","outcome":"passed","observed_at":"2026-11-23T12:00:00Z","os_version":"Linux 6.8.0","architecture":"amd64","cpu_count":8,"memory_total_bytes":17179869184,"accelerator_identity":"none","browser":"Chromium","browser_version":"150","device_identity":"host-1","provider":"whisper-stt","model_id":"whisper-small","engine_version":"1.0","model_revision":"sha256:model","product_path":"mic to API","product_path_status":"passed","artifact_ref":"receipt-1","artifact_sha256":"` + strings.Repeat("b", 64) + `"`
	tests := []struct{ name, receipts, want string }{
		{"placeholder device identity", `{"schema_version":1,"receipts":[{` + strings.Replace(base, `"device_identity":"host-1"`, `"device_identity":"unknown"`, 1) + `}]}`, "placeholder device"},
		{"placeholder browser version", `{"schema_version":1,"receipts":[{` + strings.Replace(base, `"browser_version":"150"`, `"browser_version":"unknown"`, 1) + `}]}`, "placeholder device"},
		{"wrong browser", `{"schema_version":1,"receipts":[{` + strings.Replace(base, `"browser":"Chromium"`, `"browser":"Firefox"`, 1) + `}]}`, "does not match declared"},
		{"invalid artifact hash", `{"schema_version":1,"receipts":[{` + strings.Replace(base, strings.Repeat("b", 64), "not-a-digest", 1) + `}]}`, "64-character hexadecimal"},
		{"missing artifact", `{"schema_version":1,"receipts":[{` + strings.Replace(base, `,"artifact_sha256":"`+strings.Repeat("b", 64)+`"`, "", 1) + `}]}`, "missing pinned provenance"},
		{"unknown cell", `{"schema_version":1,"receipts":[{"cell_id":"D9/unknown","outcome":"unavailable","reason":"no device"}]}`, "undeclared cell"},
		{"duplicate cell", `{"schema_version":1,"receipts":[{"cell_id":"D1-linux/batch","outcome":"unavailable","reason":"one"},{"cell_id":"D1-linux/batch","outcome":"unavailable","reason":"two"}]}`, "duplicate"},
		{"unknown field", `{"schema_version":1,"receipts":[{"cell_id":"D1-linux/batch","outcome":"unavailable","reason":"no device","qualified":true}]}`, "unknown field"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := evaluateDeviceMatrix(declaration, []byte(test.receipts))
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestInventoryOnlyBoundaryCannotQualifyAReceipt(t *testing.T) {
	declaration := deviceMatrixDeclaration{
		SchemaVersion: 1, Boundary: "inventory_only",
		Targets:  []matrixTarget{{ID: "batch", Provider: "whisper-stt", ModelID: "whisper-small"}},
		Profiles: []matrixProfile{{ID: "D1-linux", OS: "Linux amd64", Browser: "Chromium"}},
	}
	_, err := evaluateDeviceMatrix(declaration, []byte(`{"schema_version":1,"receipts":[{"cell_id":"D1-linux/batch","outcome":"passed"}]}`))
	require.ErrorContains(t, err, "forbidden by inventory_only")
}
