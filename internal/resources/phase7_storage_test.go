package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase7ResourceShellFilesAvoidRepoLocalStorageFallbacks(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cases := []struct {
		path      string
		forbidden []string
	}{
		{
			path: "resources/neo4j/config/defaults.sh",
			forbidden: []string{
				"${var_ROOT_DIR}/data/resources/neo4j",
			},
		},
		{
			path: "resources/sagemath/config/defaults.sh",
			forbidden: []string{
				"${VROOLI_ROOT:-${HOME}/Vrooli}/data",
				"${var_DATA_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}/data}",
			},
		},
		{
			path: "resources/sagemath/lib/common.sh",
			forbidden: []string{
				"${APP_ROOT}/data/resources/sagemath",
			},
		},
		{
			path: "resources/sagemath/lib/export.sh",
			forbidden: []string{
				"/home/matthalloran8/Vrooli/data/resources/sagemath/outputs",
			},
		},
		{
			path: "resources/opencode/lib/common.sh",
			forbidden: []string{
				"${APP_ROOT}/data",
				"${var_ROOT_DIR:-${APP_ROOT}}/data/credentials/openrouter-credentials.json",
			},
		},
		{
			path: "resources/litellm/config/defaults.sh",
			forbidden: []string{
				"${var_DATA_DIR}/resources/litellm",
			},
		},
		{
			path: "resources/home-assistant/config/defaults.sh",
			forbidden: []string{
				"${var_DATA_DIR}/resources/home-assistant",
			},
		},
		{
			path: "resources/questdb/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.questdb/data",
				"${HOME}/.questdb/config",
				"${HOME}/.questdb/logs",
				"${HOME}/.questdb/state",
			},
		},
		{
			path: "resources/redis/resource.json",
			forbidden: []string{
				"${HOME}/.vrooli/redis/data",
			},
		},
		{
			path: "resources/vault/resource.json",
			forbidden: []string{
				"${HOME}/.vault/data",
				"${HOME}/.vault/config",
				"${HOME}/.vault/logs",
			},
		},
		{
			path: "resources/postgres/resource.json",
			forbidden: []string{
				"${HOME}/.vrooli/postgres/main/data",
			},
		},
		{
			path: "resources/vault/config/defaults.sh",
			forbidden: []string{
				"${var_DATA_DIR}/resources/vault",
				"${HOME}/.vault/data",
				"${HOME}/.vault/config",
				"${HOME}/.vault/logs",
			},
		},
		{
			path: "resources/redis/lib/backup.sh",
			forbidden: []string{
				"${HOME}/.vrooli/redis/backups",
			},
		},
		{
			path: "resources/postgres/config/defaults.sh",
			forbidden: []string{
				"${POSTGRES_PROJECT_ROOT}/.vrooli/backups/postgres",
				"${APP_ROOT}/resources/postgres/instances",
				"${POSTGRES_PROJECT_ROOT}/.vrooli/postgres",
			},
		},
		{
			path: "resources/postgres/lib/status.sh",
			forbidden: []string{
				"${VROOLI_ROOT:-${HOME}/Vrooli}/resources/postgres/instances",
				"${var_SCRIPTS_RESOURCES_DIR}/storage/postgres",
			},
		},
		{
			path: "resources/postgres/lib/docker.sh",
			forbidden: []string{
				"${APP_ROOT}/resources/postgres/instances/",
			},
		},
		{
			path: "resources/postgres/lib/client-setup.sh",
			forbidden: []string{
				"~/.vrooli/backups/postgres/",
			},
		},
		{
			path: "resources/questdb/lib/install.sh",
			forbidden: []string{
				"${HOME}/.questdb",
			},
		},
		{
			path: "resources/questdb/lib/status.sh",
			forbidden: []string{
				"/root/.questdb",
			},
		},
		{
			path: "resources/questdb/test/integration-test.sh",
			forbidden: []string{
				"${HOME}/.questdb/data",
			},
		},
		{
			path: "resources/vault/docker/docker-compose.yml",
			forbidden: []string{
				"$HOME/.vault/data",
				"$HOME/.vault/config",
				"$HOME/.vault/logs",
			},
		},
		{
			path: "resources/postgres/examples/package-deployment.sh",
			forbidden: []string{
				"${APP_ROOT}/resources/postgres/config/database.env",
				`DATA_DIR="\$ROOT_DIR/data"`,
			},
		},
		{
			path: "resources/browserless/resource.json",
			forbidden: []string{
				"${HOME}/.browserless",
			},
		},
		{
			path: "resources/browserless/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.browserless",
			},
		},
		{
			path: "resources/browserless/lib/common.sh",
			forbidden: []string{
				"$HOME/.vrooli/browserless",
			},
		},
		{
			path: "resources/browserless/lib/browser-ops-persistent.sh",
			forbidden: []string{
				"${HOME}/.vrooli/browserless",
			},
		},
		{
			path: "resources/browserless/lib/browser-ops-stateful.sh",
			forbidden: []string{
				"${HOME}/.vrooli/browserless",
			},
		},
		{
			path: "resources/comfyui/resource.json",
			forbidden: []string{
				"${HOME}/.comfyui/models",
				"${HOME}/.comfyui/custom_nodes",
				"${HOME}/.comfyui/outputs",
				"${HOME}/.comfyui/input",
				"${HOME}/.comfyui/workflows",
				"${HOME}/.comfyui/user",
			},
		},
		{
			path: "resources/comfyui/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.comfyui",
			},
		},
		{
			path: "resources/comfyui/cli.sh",
			forbidden: []string{
				"${HOME}/.comfyui",
			},
		},
		{
			path: "resources/comfyui/test/integration-test.sh",
			forbidden: []string{
				"${HOME}/.comfyui/models/checkpoints",
			},
		},
		{
			path: "resources/comfyui/lib/test_workflow.sh",
			forbidden: []string{
				"${HOME}/.comfyui/models/checkpoints",
			},
		},
		{
			path: "resources/comfyui/examples/README.md",
			forbidden: []string{
				"${HOME}/.comfyui/workflows",
			},
		},
		{
			path: "resources/comfyui/examples/composite_comic_panels.py",
			forbidden: []string{
				"~/.comfyui/outputs",
				"~/.comfyui/workflows",
			},
		},
		{
			path: "resources/minio/resource.json",
			forbidden: []string{
				"${HOME}/.minio/data",
				"${HOME}/.minio/config",
			},
		},
		{
			path: "resources/minio/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.minio/data",
				"${HOME}/.minio/config",
			},
		},
		{
			path: "resources/minio/lib/api.sh",
			forbidden: []string{
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/lib/backup.sh",
			forbidden: []string{
				"${HOME}/.minio/backups",
				"${HOME}/.minio/data",
				"${HOME}/.minio/config",
			},
		},
		{
			path: "resources/minio/lib/core.sh",
			forbidden: []string{
				"${HOME}/.minio",
				"${HOME}/.minio/data",
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/lib/performance.sh",
			forbidden: []string{
				"${HOME}/.minio/config/performance.conf",
				"${HOME}/.minio/config/performance.profile",
			},
		},
		{
			path: "resources/minio/lib/docker.sh",
			forbidden: []string{
				"${HOME}/.minio/config/performance.conf",
				"${HOME}/.minio/cache",
			},
		},
		{
			path: "resources/minio/lib/replication.sh",
			forbidden: []string{
				"$HOME/.minio/config",
			},
		},
		{
			path: "resources/minio/lib/test.sh",
			forbidden: []string{
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/test/phases/test-integration.sh",
			forbidden: []string{
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/test/phases/test-unit.sh",
			forbidden: []string{
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/examples/n8n-workflow-storage.sh",
			forbidden: []string{
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/examples/ollama-model-cache.sh",
			forbidden: []string{
				"${HOME}/.ollama/models",
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/minio/examples/postgres-backup.sh",
			forbidden: []string{
				"${HOME}/.minio/config/credentials",
			},
		},
		{
			path: "resources/ollama/resource.json",
			forbidden: []string{
				"${HOME}/.ollama",
			},
		},
		{
			path: "resources/ollama/lib/api.sh",
			forbidden: []string{
				"~/.ollama/models",
			},
		},
		{
			path: "resources/qdrant/resource.json",
			forbidden: []string{
				"${HOME}/.qdrant/data",
				"${HOME}/.qdrant/snapshots",
			},
		},
		{
			path: "resources/qdrant/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.qdrant/data",
				"${HOME}/.qdrant/config",
				"${HOME}/.qdrant/snapshots",
			},
		},
		{
			path: "resources/qdrant/lib/error-handler.sh",
			forbidden: []string{
				"$HOME/.qdrant/errors.log",
				"$HOME/.qdrant/error-metrics.json",
				"$HOME/.qdrant/failed-operations",
			},
		},
		{
			path: "resources/qdrant/monitoring/health-check.sh",
			forbidden: []string{
				"$HOME/.qdrant/health.log",
			},
		},
		{
			path: "resources/qdrant/lib/backup.sh",
			forbidden: []string{
				"$HOME/.vrooli/backups",
			},
		},
		{
			path: "resources/qdrant/test/integration-test.sh",
			forbidden: []string{
				"${HOME}/.qdrant/data",
			},
		},
		{
			path: "resources/qdrant/scripts/monitoring-dashboard.sh",
			forbidden: []string{
				"$HOME/.qdrant/error-metrics.json",
				"$HOME/.qdrant/failed-operations",
				"$HOME/.qdrant/identity/vrooli-main.json",
				"$HOME/.qdrant/health.log",
				"$HOME/.qdrant/errors.log",
			},
		},
		{
			path: "resources/searxng/resource.json",
			forbidden: []string{
				"${HOME}/.searxng",
			},
		},
		{
			path: "resources/searxng/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.searxng",
			},
		},
		{
			path: "resources/searxng/docker/docker-compose.yml",
			forbidden: []string{
				"${HOME}/.searxng",
			},
		},
		{
			path: "resources/searxng/lib/install.sh",
			forbidden: []string{
				"${HOME}/.searxng-backup-",
			},
		},
		{
			path: "resources/searxng/lib/api.bats",
			forbidden: []string{
				"$HOME/.searxng",
			},
		},
		{
			path: "resources/whisper/config/defaults.sh",
			forbidden: []string{
				"${HOME}/.whisper",
			},
		},
		{
			path: "resources/whisper/docker/docker-compose.yml",
			forbidden: []string{
				"$HOME/.whisper/models",
				"$HOME/.whisper/uploads",
			},
		},
		{
			path: "resources/whisper/docker/docker-compose.gpu.yml",
			forbidden: []string{
				"$HOME/.whisper/models",
				"$HOME/.whisper/uploads",
			},
		},
		{
			path: "resources/whisper/test/integration-test.sh",
			forbidden: []string{
				"${HOME}/.whisper/models",
			},
		},
		{
			path: "resources/whisper/inject.sh",
			forbidden: []string{
				"${HOME}/.whisper",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.path)))
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			content := string(body)
			for _, forbidden := range tc.forbidden {
				if strings.Contains(content, forbidden) {
					t.Fatalf("%s still contains forbidden storage fallback %q", tc.path, forbidden)
				}
			}
		})
	}
}
