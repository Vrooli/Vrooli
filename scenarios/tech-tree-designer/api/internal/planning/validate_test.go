package planning

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompilerValidatorValidPlannedProtoPasses(t *testing.T) {
	validator := NewCompilerValidator(t.TempDir())
	findings, err := validator.Validate(context.Background(), Scenario{
		Slug: "planned-demo",
		Files: []ProtoFile{{
			Path: "planned-demo/v1/api/service.proto",
			Text: validProtoText(),
		}},
	})

	require.NoError(t, err)
	require.Empty(t, errorCodes(findings))
}

func TestCompilerValidatorResolvesLiveImportsWithWellKnownDependencies(t *testing.T) {
	root := t.TempDir()
	commonDir := filepath.Join(root, "common/v1")
	require.NoError(t, os.MkdirAll(commonDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(commonDir, "types.proto"), []byte(`syntax = "proto3";

package common.v1;

import "google/protobuf/struct.proto";

option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1;commonv1";

// @stability stable
message JsonValue {
  google.protobuf.Value value = 1;
}
`), 0o644))

	validator := NewCompilerValidator(root)
	findings, err := validator.Validate(context.Background(), Scenario{
		Slug: "planned-demo",
		Files: []ProtoFile{{
			Path: "planned-demo/v1/api/service.proto",
			Text: `syntax = "proto3";

package planned_demo.v1.api;

import "common/v1/types.proto";

option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/planned-demo/v1/api;api_v1";

// @stability experimental
message PlannedThing {
  string thing_id = 1;
  common.v1.JsonValue metadata = 2;
}
`,
		}},
	})

	require.NoError(t, err)
	require.Empty(t, errorCodes(findings))
}

func TestCompilerValidatorReportsConventionAndCompileFindings(t *testing.T) {
	root := t.TempDir()
	validator := NewCompilerValidator(root)
	findings, err := validator.Validate(context.Background(), Scenario{
		Slug: "planned-demo",
		Files: []ProtoFile{{
			Path: "planned-demo/v1/api/service.proto",
			Text: `syntax = "proto3";
package planned_demo.v1.api;
option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/planned-demo/v1/api;api_v1";

import "missing/v1/api/service.proto";

message bad_name {
  string badID = 1;
}
`,
		}},
	})

	require.NoError(t, err)
	codes := errorCodes(findings)
	require.Contains(t, codes, "missing_stability")
	require.Contains(t, codes, "unresolved_import")
	require.Contains(t, codes, "message_name_not_pascal_case")
	require.Contains(t, codes, "field_name_not_snake_case")
	require.Contains(t, codes, "compile_failed")
	require.NoFileExists(t, filepath.Join(root, "planned-demo/v1/api/service.proto"), "validator must not mutate schemas root")
}

func validProtoText() string {
	return `syntax = "proto3";

package planned_demo.v1.api;

option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/planned-demo/v1/api;api_v1";

// @stability experimental
message PlannedThing {
  string thing_id = 1;
  string created_at = 2;
}
`
}

func errorCodes(findings []PlanFinding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.Code)
	}
	return out
}
