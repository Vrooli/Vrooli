package resources

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apicoresecrets "github.com/vrooli/api-core/secrets"
	repocontract "github.com/vrooli/repo-contract-go"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/secrets"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
)

func TestLoadPortRegistryReadsTypedJSON(t *testing.T) {
	root := t.TempDir()
	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{"db": "5432-5499"},
	})

	registry, err := LoadPortRegistry(root)
	if err != nil {
		t.Fatalf("LoadPortRegistry: %v", err)
	}
	if got := registry.ResourcePorts["postgres"]; got != 5433 {
		t.Fatalf("postgres port = %d, want 5433", got)
	}
	if got := registry.ReservedRanges["db"]; got != "5432-5499" {
		t.Fatalf("reserved range = %q, want 5432-5499", got)
	}
}

func TestLoadResourceEnvironmentUsesTypedDefaultsAndSecrets(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"browserless": 4110, "postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	testresource.WriteResourceManifest(t, root, "postgres", manifestpkg.ResourceManifest{
		Name:            "postgres",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "postgresql", Container: 5432, Host: 5433}},
		Runtime: manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
			Env: map[string]string{
				"POSTGRES_DB":   "vrooli",
				"POSTGRES_USER": "vrooli",
			},
		},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"POSTGRES_HOST": "localhost", "POSTGRES_SSLMODE": "disable"},
			FromPorts:      map[string]string{"POSTGRES_PORT": "postgresql"},
			FromRuntimeEnv: []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"POSTGRES_URL": {Template: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE}"},
				"DATABASE_URL": {Template: "${POSTGRES_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "browserless", manifestpkg.ResourceManifest{
		Name:            "browserless",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 3000, Host: 4110}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "browserless/chrome:stable"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"BROWSERLESS_HOST": "localhost"},
			FromPorts:      map[string]string{"BROWSERLESS_PORT": "http"},
			FromRuntimeEnv: []string{"BROWSERLESS_TOKEN"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"BROWSERLESS_URL":      {Template: "http://${BROWSERLESS_HOST}:${BROWSERLESS_PORT}"},
				"BROWSERLESS_BASE_URL": {Template: "${BROWSERLESS_URL}"},
			},
		},
	})
	t.Setenv(secrets.KeyEnvVar, "metadata-test-secret-key")
	store := secrets.NewUserStore(home)
	if err := store.Save(map[string]string{
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_USER":     "vrooli",
		"BROWSERLESS_TOKEN": "abc123",
	}); err != nil {
		t.Fatalf("Save encrypted secrets: %v", err)
	}

	postgresEnv, err := LoadResourceEnvironment(root, home, "postgres")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(postgres): %v", err)
	}
	if got := postgresEnv["POSTGRES_PORT"]; got != "5433" {
		t.Fatalf("POSTGRES_PORT = %q, want 5433", got)
	}
	if got := postgresEnv["POSTGRES_HOST"]; got != "localhost" {
		t.Fatalf("POSTGRES_HOST = %q, want localhost", got)
	}
	if got := postgresEnv["POSTGRES_PASSWORD"]; got != "secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want secret", got)
	}

	browserlessEnv, err := LoadResourceEnvironment(root, home, "browserless")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(browserless): %v", err)
	}
	if got := browserlessEnv["BROWSERLESS_PORT"]; got != "4110" {
		t.Fatalf("BROWSERLESS_PORT = %q, want 4110", got)
	}
	if got := browserlessEnv["BROWSERLESS_BASE_URL"]; got != "http://localhost:4110" {
		t.Fatalf("BROWSERLESS_BASE_URL = %q", got)
	}
	if got := browserlessEnv["BROWSERLESS_URL"]; got != "http://localhost:4110" {
		t.Fatalf("BROWSERLESS_URL = %q, want http://localhost:4110", got)
	}
	if got := browserlessEnv["BROWSERLESS_TOKEN"]; got != "abc123" {
		t.Fatalf("BROWSERLESS_TOKEN = %q, want abc123", got)
	}
}

func TestLoadResourceEnvironmentUsesEncryptedSecrets(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	writePostgresManifestFixture(t, root)

	t.Setenv(secrets.KeyEnvVar, "resource-secret-key")
	store := secrets.NewUserStore(home)
	if err := store.Save(map[string]string{
		"POSTGRES_PASSWORD": "encrypted-secret",
		"POSTGRES_USER":     "vrooli",
	}); err != nil {
		t.Fatalf("Save encrypted secrets: %v", err)
	}

	postgresEnv, err := LoadResourceEnvironment(root, home, "postgres")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(postgres): %v", err)
	}
	if got := postgresEnv["POSTGRES_PASSWORD"]; got != "encrypted-secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want encrypted-secret", got)
	}
	if got := postgresEnv["POSTGRES_USER"]; got != "vrooli" {
		t.Fatalf("POSTGRES_USER = %q, want vrooli", got)
	}
}

func TestLoadResourceEnvironmentPrefersSecretsOverRuntimePlaceholders(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	writeEnvManifestFixture(t, root, "postgres", manifestpkg.ResourceManifest{
		Name:            "postgres",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "postgresql", Container: 5432, Host: 5433}},
		Runtime: manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
			Env: map[string]string{
				"POSTGRES_DB":       "vrooli",
				"POSTGRES_USER":     "vrooli",
				"POSTGRES_PASSWORD": "placeholder",
			},
		},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"POSTGRES_HOST": "localhost", "POSTGRES_SSLMODE": "disable"},
			FromPorts:      map[string]string{"POSTGRES_PORT": "postgresql"},
			FromRuntimeEnv: []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"POSTGRES_URL": {Template: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE}"},
				"DATABASE_URL": {Template: "${POSTGRES_URL}"},
			},
		},
	})

	t.Setenv(secrets.KeyEnvVar, "metadata-prefers-encrypted")
	store := secrets.NewUserStore(home)
	if err := store.Save(map[string]string{
		"POSTGRES_PASSWORD": "real-secret",
	}); err != nil {
		t.Fatalf("Save encrypted secrets: %v", err)
	}

	postgresEnv, err := LoadResourceEnvironment(root, home, "postgres")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(postgres): %v", err)
	}
	if got := postgresEnv["POSTGRES_PASSWORD"]; got != "real-secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want real-secret", got)
	}
	if got := postgresEnv["DATABASE_URL"]; !strings.Contains(got, ":real-secret@") {
		t.Fatalf("DATABASE_URL = %q, want runtime password from secrets", got)
	}
}

func TestLoadResourceEnvironmentFallsBackToPlaintextUserSecrets(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	writePostgresManifestFixture(t, root)

	store, err := apicoresecrets.NewUserStore(apicoresecrets.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	if err := store.Save(map[string]string{
		"POSTGRES_PASSWORD": "plaintext-secret",
		"POSTGRES_USER":     "vrooli",
	}); err != nil {
		t.Fatalf("Save plaintext secrets: %v", err)
	}

	postgresEnv, err := LoadResourceEnvironment(root, home, "postgres")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(postgres): %v", err)
	}
	if got := postgresEnv["POSTGRES_PASSWORD"]; got != "plaintext-secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want plaintext-secret", got)
	}
	if got := postgresEnv["DATABASE_URL"]; !strings.Contains(got, ":plaintext-secret@") {
		t.Fatalf("DATABASE_URL = %q, want plaintext user secret", got)
	}
}

func TestLoadResourceEnvironmentPrefersEncryptedSecretsOverPlaintextUserSecrets(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	writePostgresManifestFixture(t, root)

	plaintextStore, err := apicoresecrets.NewUserStore(apicoresecrets.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	if err := plaintextStore.Save(map[string]string{
		"POSTGRES_PASSWORD": "plaintext-secret",
		"POSTGRES_USER":     "vrooli",
	}); err != nil {
		t.Fatalf("Save plaintext secrets: %v", err)
	}

	t.Setenv(secrets.KeyEnvVar, "metadata-prefers-encrypted-over-plaintext")
	encryptedStore := secrets.NewUserStore(home)
	if err := encryptedStore.Save(map[string]string{
		"POSTGRES_PASSWORD": "encrypted-secret",
	}); err != nil {
		t.Fatalf("Save encrypted secrets: %v", err)
	}

	postgresEnv, err := LoadResourceEnvironment(root, home, "postgres")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(postgres): %v", err)
	}
	if got := postgresEnv["POSTGRES_PASSWORD"]; got != "encrypted-secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want encrypted-secret", got)
	}
	if got := postgresEnv["DATABASE_URL"]; !strings.Contains(got, ":encrypted-secret@") {
		t.Fatalf("DATABASE_URL = %q, want encrypted user secret", got)
	}
}

func TestLoadResourceEnvironmentSupportsNativeDerivedURLs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts: map[string]int{
			"comfyui":         8188,
			"minio":           9000,
			"redis":           6380,
			"qdrant":          6333,
			"questdb":         9009,
			"ollama":          11434,
			"vault":           8200,
			"searxng":         8280,
			"unstructured-io": 11450,
		},
		ReservedRanges: map[string]string{},
	})
	writeEnvManifestFixture(t, root, "redis", manifestpkg.ResourceManifest{
		Name:            "redis",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "redis", Container: 6379, Host: 6380}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "redis:7-alpine"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"REDIS_HOST": "localhost", "REDIS_DB": "0"},
			FromPorts:      map[string]string{"REDIS_PORT": "redis"},
			FromRuntimeEnv: []string{"REDIS_PASSWORD"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"REDIS_URL":      {Template: "redis://${REDIS_HOST}:${REDIS_PORT}"},
				"REDIS_BASE_URL": {Template: "${REDIS_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "qdrant", manifestpkg.ResourceManifest{
		Name:            "qdrant",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports: []manifestpkg.ResourcePort{
			{Name: "http", Container: 6333, Host: 6333},
			{Name: "grpc", Container: 6334, Host: 6334},
		},
		Runtime: manifestpkg.ResourceRuntime{Image: "qdrant/qdrant:latest"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"QDRANT_HOST": "localhost"},
			FromPorts: map[string]string{"QDRANT_PORT": "http", "QDRANT_GRPC_PORT": "grpc"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"QDRANT_URL":      {Template: "http://${QDRANT_HOST}:${QDRANT_PORT}"},
				"QDRANT_BASE_URL": {Template: "${QDRANT_URL}"},
				"QDRANT_GRPC_URL": {Template: "grpc://${QDRANT_HOST}:${QDRANT_GRPC_PORT}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "ollama", manifestpkg.ResourceManifest{
		Name:            "ollama",
		Driver:          "docker-service",
		PortabilityTier: "partial",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 11434, Host: 11434}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "ollama/ollama:latest"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"OLLAMA_HOST": "localhost"},
			FromPorts: map[string]string{"OLLAMA_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"OLLAMA_URL":      {Template: "http://${OLLAMA_HOST}:${OLLAMA_PORT}"},
				"OLLAMA_BASE_URL": {Template: "${OLLAMA_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "comfyui", manifestpkg.ResourceManifest{
		Name:            "comfyui",
		Driver:          "docker-service",
		PortabilityTier: "partial",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 8188, Host: 8188}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "zhangp365/comfyui:latest"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"COMFYUI_HOST": "localhost"},
			FromPorts: map[string]string{"COMFYUI_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"COMFYUI_URL":      {Template: "http://${COMFYUI_HOST}:${COMFYUI_PORT}"},
				"COMFYUI_BASE_URL": {Template: "${COMFYUI_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "minio", manifestpkg.ResourceManifest{
		Name:            "minio",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports: []manifestpkg.ResourcePort{
			{Name: "api", Container: 9000, Host: 9000},
			{Name: "console", Container: 9001, Host: 9001},
		},
		Runtime: manifestpkg.ResourceRuntime{
			Image: "minio/minio:latest",
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
		},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"MINIO_HOST": "localhost"},
			FromPorts:      map[string]string{"MINIO_PORT": "api", "MINIO_CONSOLE_PORT": "console"},
			FromRuntimeEnv: []string{"MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"MINIO_URL":         {Template: "http://${MINIO_HOST}:${MINIO_PORT}"},
				"MINIO_ENDPOINT":    {Template: "${MINIO_HOST}:${MINIO_PORT}"},
				"MINIO_ACCESS_KEY":  {Template: "${MINIO_ROOT_USER}"},
				"MINIO_SECRET_KEY":  {Template: "${MINIO_ROOT_PASSWORD}"},
				"MINIO_CONSOLE_URL": {Template: "http://${MINIO_HOST}:${MINIO_CONSOLE_PORT}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "vault", manifestpkg.ResourceManifest{
		Name:            "vault",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 8200, Host: 8200}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "hashicorp/vault:1.17"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"VAULT_HOST": "localhost", "VAULT_TOKEN": "myroot"},
			FromPorts: map[string]string{"VAULT_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"VAULT_URL":  {Template: "http://${VAULT_HOST}:${VAULT_PORT}"},
				"VAULT_ADDR": {Template: "${VAULT_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "questdb", manifestpkg.ResourceManifest{
		Name:            "questdb",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports: []manifestpkg.ResourcePort{
			{Name: "http", Container: 9000, Host: 9009},
			{Name: "postgresql", Container: 8812, Host: 8812},
			{Name: "influxdb-line", Container: 9009, Host: 9011},
		},
		Runtime: manifestpkg.ResourceRuntime{
			Image: "questdb/questdb:8.1.2",
			Env: map[string]string{
				"QDB_PG_USER":     "admin",
				"QDB_PG_PASSWORD": "quest",
			},
		},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static: map[string]string{
				"QUESTDB_HOST":        "localhost",
				"QUESTDB_PG_USER":     "admin",
				"QUESTDB_PG_PASSWORD": "quest",
			},
			FromPorts: map[string]string{
				"QUESTDB_PORT":      "http",
				"QUESTDB_HTTP_PORT": "http",
				"QUESTDB_PG_PORT":   "postgresql",
				"QUESTDB_ILP_PORT":  "influxdb-line",
			},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"QUESTDB_URL":      {Template: "http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}"},
				"QUESTDB_BASE_URL": {Template: "${QUESTDB_URL}"},
				"QUESTDB_PG_URL":   {Template: "postgresql://${QUESTDB_PG_USER}:${QUESTDB_PG_PASSWORD}@${QUESTDB_HOST}:${QUESTDB_PG_PORT}/qdb"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "searxng", manifestpkg.ResourceManifest{
		Name:            "searxng",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 8080, Host: 8280}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "searxng/searxng:latest"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"SEARXNG_SERVICE_HOST": "localhost"},
			FromPorts: map[string]string{"SEARXNG_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"SEARXNG_URL":  {Template: "http://${SEARXNG_SERVICE_HOST}:${SEARXNG_PORT}"},
				"SEARXNG_HOST": {Template: "${SEARXNG_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "unstructured-io", manifestpkg.ResourceManifest{
		Name:            "unstructured-io",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 8000, Host: 11450}},
		Runtime:         manifestpkg.ResourceRuntime{Image: "downloads.unstructured.io/unstructured-api:latest"},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"UNSTRUCTURED_HOST": "localhost"},
			FromPorts: map[string]string{"UNSTRUCTURED_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"UNSTRUCTURED_URL":    {Template: "http://${UNSTRUCTURED_HOST}:${UNSTRUCTURED_PORT}"},
				"UNSTRUCTURED_IO_URL": {Template: "${UNSTRUCTURED_URL}"},
			},
		},
	})

	redisEnv, err := LoadResourceEnvironment(root, home, "redis")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(redis): %v", err)
	}
	if got := redisEnv["REDIS_URL"]; got != "redis://localhost:6380" {
		t.Fatalf("REDIS_URL = %q", got)
	}
	qdrantEnv, err := LoadResourceEnvironment(root, home, "qdrant")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(qdrant): %v", err)
	}
	if got := qdrantEnv["QDRANT_GRPC_URL"]; got != "grpc://localhost:6334" {
		t.Fatalf("QDRANT_GRPC_URL = %q", got)
	}
	ollamaEnv, err := LoadResourceEnvironment(root, home, "ollama")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(ollama): %v", err)
	}
	if got := ollamaEnv["OLLAMA_URL"]; got != "http://localhost:11434" {
		t.Fatalf("OLLAMA_URL = %q", got)
	}
	comfyuiEnv, err := LoadResourceEnvironment(root, home, "comfyui")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(comfyui): %v", err)
	}
	if got := comfyuiEnv["COMFYUI_BASE_URL"]; got != "http://localhost:8188" {
		t.Fatalf("COMFYUI_BASE_URL = %q", got)
	}
	minioEnv, err := LoadResourceEnvironment(root, home, "minio")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(minio): %v", err)
	}
	if got := minioEnv["MINIO_ENDPOINT"]; got != "localhost:9000" {
		t.Fatalf("MINIO_ENDPOINT = %q", got)
	}
	vaultEnv, err := LoadResourceEnvironment(root, home, "vault")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(vault): %v", err)
	}
	if got := vaultEnv["VAULT_ADDR"]; got != "http://localhost:8200" {
		t.Fatalf("VAULT_ADDR = %q", got)
	}
	questdbEnv, err := LoadResourceEnvironment(root, home, "questdb")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(questdb): %v", err)
	}
	if got := questdbEnv["QUESTDB_URL"]; got != "http://localhost:9009" {
		t.Fatalf("QUESTDB_URL = %q", got)
	}
	if got := questdbEnv["QUESTDB_PG_URL"]; got != "postgresql://admin:quest@localhost:8812/qdb" {
		t.Fatalf("QUESTDB_PG_URL = %q", got)
	}
	searxngEnv, err := LoadResourceEnvironment(root, home, "searxng")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(searxng): %v", err)
	}
	if got := searxngEnv["SEARXNG_HOST"]; got != "http://localhost:8280" {
		t.Fatalf("SEARXNG_HOST = %q", got)
	}
	unstructuredEnv, err := LoadResourceEnvironment(root, home, "unstructured-io")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(unstructured-io): %v", err)
	}
	if got := unstructuredEnv["UNSTRUCTURED_IO_URL"]; got != "http://localhost:11450" {
		t.Fatalf("UNSTRUCTURED_IO_URL = %q", got)
	}
}

func TestLoadResourceEnvironmentSupportsNativeNonDockerContracts(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeEnvManifestFixture(t, root, "fixturecli", manifestpkg.ResourceManifest{
		Name:            "fixturecli",
		Driver:          "external-cli",
		Binary:          "resource-fixturecli",
		PortabilityTier: "full",
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static: map[string]string{
				"FIXTURECLI_PATH":         "resource-fixturecli",
				"FIXTURECLI_JOURNAL_MODE": "WAL",
			},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"FIXTURECLI_DATA_PATH":  {Template: "${RESOURCE_DATA_DIR}/data"},
				"FIXTURECLI_STATE_PATH": {Template: "${RESOURCE_STATE_DIR}/state"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "openrouter", manifestpkg.ResourceManifest{
		Name:            "openrouter",
		Driver:          "cloud-api",
		Endpoint:        "https://openrouter.ai/api/v1/models",
		PortabilityTier: "full",
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"OPENROUTER_API_BASE": "https://openrouter.ai/api/v1"},
			FromRuntimeEnv: []string{"OPENROUTER_API_KEY"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"OPENROUTER_URL":          {Template: "${OPENROUTER_API_BASE}"},
				"RESOURCE_OPENROUTER_URL": {Template: "${OPENROUTER_API_BASE}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "home-assistant", manifestpkg.ResourceManifest{
		Name:            "home-assistant",
		Driver:          "compose-service",
		ComposeFile:     "compose.yaml",
		PortabilityTier: "partial",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 8123, Host: 8123}},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"HOME_ASSISTANT_HOST": "localhost", "HOME_ASSISTANT_CONTAINER_NAME": "home-assistant"},
			FromPorts:      map[string]string{"HOME_ASSISTANT_PORT": "http"},
			FromRuntimeEnv: []string{"HOME_ASSISTANT_TOKEN"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"HOME_ASSISTANT_BASE_URL": {Template: "http://${HOME_ASSISTANT_HOST}:${HOME_ASSISTANT_PORT}"},
				"HOME_ASSISTANT_URL":      {Template: "${HOME_ASSISTANT_BASE_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "whisper", manifestpkg.ResourceManifest{
		Name:            "whisper",
		Driver:          "compose-service",
		ComposeFile:     "docker/docker-compose.yml",
		PortabilityTier: "partial",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 9000, Host: 8090}},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"WHISPER_HOST": "localhost"},
			FromPorts: map[string]string{"WHISPER_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"WHISPER_URL":      {Template: "http://${WHISPER_HOST}:${WHISPER_PORT}"},
				"WHISPER_BASE_URL": {Template: "${WHISPER_URL}"},
				"WHISPER_STT_URL":  {Template: "${WHISPER_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "kokoro", manifestpkg.ResourceManifest{
		Name:            "kokoro",
		Driver:          "compose-service",
		ComposeFile:     "docker/docker-compose.yml",
		PortabilityTier: "partial",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 8880, Host: 8880}},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:    map[string]string{"KOKORO_HOST": "localhost"},
			FromPorts: map[string]string{"KOKORO_PORT": "http"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"KOKORO_URL":      {Template: "http://${KOKORO_HOST}:${KOKORO_PORT}"},
				"KOKORO_BASE_URL": {Template: "${KOKORO_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "judge0", manifestpkg.ResourceManifest{
		Name:            "judge0",
		Driver:          "compose-service",
		ComposeFile:     "compose.yaml",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "http", Container: 2358, Host: 2358}},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"JUDGE0_HOST": "localhost", "JUDGE0_API_PREFIX": "/api/v1"},
			FromPorts:      map[string]string{"JUDGE0_PORT": "http"},
			FromRuntimeEnv: []string{"JUDGE0_API_KEY"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"JUDGE0_BASE_URL": {Template: "http://${JUDGE0_HOST}:${JUDGE0_PORT}"},
				"JUDGE0_URL":      {Template: "${JUDGE0_BASE_URL}"},
			},
		},
	})
	writeEnvManifestFixture(t, root, "claude-code", manifestpkg.ResourceManifest{
		Name:            "claude-code",
		Driver:          "external-cli",
		Binary:          "claude",
		PortabilityTier: "partial",
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static: map[string]string{
				"CLAUDE_CODE_PATH": "claude",
				"CLAUDE_CODE_URL":  "http://localhost:8100",
			},
		},
	})
	writeEnvManifestFixture(t, root, "codex", manifestpkg.ResourceManifest{
		Name:            "codex",
		Driver:          "external-cli",
		Binary:          "codex",
		PortabilityTier: "partial",
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static: map[string]string{"CODEX_PATH": "codex"},
		},
	})
	writeEnvManifestFixture(t, root, "opencode", manifestpkg.ResourceManifest{
		Name:            "opencode",
		Driver:          "external-cli",
		Binary:          "opencode",
		PortabilityTier: "partial",
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static: map[string]string{"OPENCODE_PATH": "opencode"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"OPENCODE_CONFIG_DIR":      {Template: "${RESOURCE_CONFIG_DIR}"},
				"OPENCODE_DATA_DIR":        {Template: "${RESOURCE_DATA_DIR}"},
				"OPENCODE_CACHE_DIR":       {Template: "${RESOURCE_CACHE_DIR}"},
				"OPENCODE_LOG_DIR":         {Template: "${RESOURCE_LOGS_DIR}"},
				"OPENCODE_STATE_DIR":       {Template: "${RESOURCE_STATE_DIR}"},
				"OPENCODE_XDG_CONFIG_HOME": {Template: "${OPENCODE_CONFIG_DIR}/xdg-config"},
				"OPENCODE_XDG_DATA_HOME":   {Template: "${OPENCODE_DATA_DIR}/xdg-data"},
			},
		},
	})
	t.Setenv(secrets.KeyEnvVar, "native-non-docker-encrypted")
	store := secrets.NewUserStore(home)
	if err := store.Save(map[string]string{
		"OPENROUTER_API_KEY":   "sk-or-test",
		"HOME_ASSISTANT_TOKEN": "ha-token",
		"JUDGE0_API_KEY":       "judge0-token",
	}); err != nil {
		t.Fatalf("Save encrypted secrets: %v", err)
	}

	fixtureEnv, err := LoadResourceEnvironment(root, home, "fixturecli")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(fixturecli): %v", err)
	}
	if got := fixtureEnv["FIXTURECLI_DATA_PATH"]; got != filepath.Join(home, ".local", "share", "vrooli", "resources", "fixturecli", "data") {
		t.Fatalf("FIXTURECLI_DATA_PATH = %q", got)
	}

	openrouterEnv, err := LoadResourceEnvironment(root, home, "openrouter")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(openrouter): %v", err)
	}
	if got := openrouterEnv["RESOURCE_OPENROUTER_URL"]; got != "https://openrouter.ai/api/v1" {
		t.Fatalf("RESOURCE_OPENROUTER_URL = %q", got)
	}
	if got := openrouterEnv["OPENROUTER_API_KEY"]; got != "sk-or-test" {
		t.Fatalf("OPENROUTER_API_KEY = %q", got)
	}

	homeAssistantEnv, err := LoadResourceEnvironment(root, home, "home-assistant")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(home-assistant): %v", err)
	}
	if got := homeAssistantEnv["HOME_ASSISTANT_URL"]; got != "http://localhost:8123" {
		t.Fatalf("HOME_ASSISTANT_URL = %q", got)
	}
	if got := homeAssistantEnv["HOME_ASSISTANT_CONTAINER_NAME"]; got != "home-assistant" {
		t.Fatalf("HOME_ASSISTANT_CONTAINER_NAME = %q", got)
	}

	whisperEnv, err := LoadResourceEnvironment(root, home, "whisper")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(whisper): %v", err)
	}
	if got := whisperEnv["WHISPER_STT_URL"]; got != "http://localhost:8090" {
		t.Fatalf("WHISPER_STT_URL = %q", got)
	}

	kokoroEnv, err := LoadResourceEnvironment(root, home, "kokoro")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(kokoro): %v", err)
	}
	if got := kokoroEnv["KOKORO_BASE_URL"]; got != "http://localhost:8880" {
		t.Fatalf("KOKORO_BASE_URL = %q", got)
	}

	judge0Env, err := LoadResourceEnvironment(root, home, "judge0")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(judge0): %v", err)
	}
	if got := judge0Env["JUDGE0_URL"]; got != "http://localhost:2358" {
		t.Fatalf("JUDGE0_URL = %q", got)
	}
	if got := judge0Env["JUDGE0_API_KEY"]; got != "judge0-token" {
		t.Fatalf("JUDGE0_API_KEY = %q", got)
	}

	claudeEnv, err := LoadResourceEnvironment(root, home, "claude-code")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(claude-code): %v", err)
	}
	if got := claudeEnv["CLAUDE_CODE_PATH"]; got != "claude" {
		t.Fatalf("CLAUDE_CODE_PATH = %q", got)
	}

	codexEnv, err := LoadResourceEnvironment(root, home, "codex")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(codex): %v", err)
	}
	if got := codexEnv["CODEX_PATH"]; got != "codex" {
		t.Fatalf("CODEX_PATH = %q", got)
	}

	opencodeEnv, err := LoadResourceEnvironment(root, home, "opencode")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(opencode): %v", err)
	}
	if got := opencodeEnv["OPENCODE_DATA_DIR"]; got != filepath.Join(home, ".local", "share", "vrooli", "resources", "opencode") {
		t.Fatalf("OPENCODE_DATA_DIR = %q", got)
	}
	if got := opencodeEnv["OPENCODE_CONFIG_DIR"]; got != filepath.Join(home, ".config", "vrooli", "resources", "opencode") {
		t.Fatalf("OPENCODE_CONFIG_DIR = %q", got)
	}
	if got := opencodeEnv["OPENCODE_CACHE_DIR"]; got != filepath.Join(home, ".cache", "vrooli", "resources", "opencode") {
		t.Fatalf("OPENCODE_CACHE_DIR = %q", got)
	}
	if got := opencodeEnv["OPENCODE_LOG_DIR"]; got != filepath.Join(home, ".local", "state", "logs", "vrooli", "resources", "opencode") {
		t.Fatalf("OPENCODE_LOG_DIR = %q", got)
	}
	if got := opencodeEnv["OPENCODE_STATE_DIR"]; got != filepath.Join(home, ".local", "state", "vrooli", "resources", "opencode") {
		t.Fatalf("OPENCODE_STATE_DIR = %q", got)
	}
	if got := opencodeEnv["OPENCODE_XDG_DATA_HOME"]; got != filepath.Join(home, ".local", "share", "vrooli", "resources", "opencode", "xdg-data") {
		t.Fatalf("OPENCODE_XDG_DATA_HOME = %q", got)
	}
}

func TestLoadResourceEnvironmentFailsWhenEncryptedSecretsRequireKey(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	writePostgresManifestFixture(t, root)
	testkitgo.WriteJSONMode(t, filepath.Join(root, ".vrooli", "secrets.json"), map[string]string{
		"POSTGRES_PASSWORD": "legacy-secret",
		"POSTGRES_USER":     "vrooli",
	}, 0o600)

	t.Setenv(secrets.KeyEnvVar, "writer-secret-key")
	store := secrets.NewUserStore(home)
	if err := store.Save(map[string]string{
		"POSTGRES_PASSWORD": "encrypted-secret",
		"POSTGRES_USER":     "vrooli",
	}); err != nil {
		t.Fatalf("Save encrypted secrets: %v", err)
	}
	t.Setenv(secrets.KeyEnvVar, "")

	_, err := LoadResourceEnvironment(root, home, "postgres")
	if err == nil || !errors.Is(err, secrets.ErrMissingKey) {
		t.Fatalf("LoadResourceEnvironment(postgres) error = %v, want ErrMissingKey", err)
	}
}

func TestLoadResourceEnvironmentFailsClosedWhenEncryptedSecretsAreInvalid(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testresource.WritePortRegistryState(t, root, PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{},
	})
	writePostgresManifestFixture(t, root)
	encryptedPath, err := repocontract.UserEncryptedSecretsPath(home)
	if err != nil {
		t.Fatalf("UserEncryptedSecretsPath: %v", err)
	}
	testkitgo.WriteRawJSON(t, encryptedPath, `{`, 0o600)

	_, err = LoadResourceEnvironment(root, home, "postgres")
	if err == nil {
		t.Fatal("expected invalid encrypted secrets to fail closed")
	}
}

func TestActualDockerServiceResourceManifestsResolveNativeExports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	skipIfRepoSecretsRequireMigrationOrKey(t, root)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home: %v", err)
	}

	comfyuiEnv, err := LoadResourceEnvironment(root, home, "comfyui")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(comfyui): %v", err)
	}
	if got := comfyuiEnv["COMFYUI_URL"]; got != "http://localhost:8188" {
		t.Fatalf("COMFYUI_URL = %q", got)
	}

	questdbEnv, err := LoadResourceEnvironment(root, home, "questdb")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(questdb): %v", err)
	}
	if got := questdbEnv["QUESTDB_URL"]; got != "http://localhost:9009" {
		t.Fatalf("QUESTDB_URL = %q", got)
	}
	if got := questdbEnv["QUESTDB_PG_URL"]; got != "postgresql://admin:quest@localhost:8812/qdb" {
		t.Fatalf("QUESTDB_PG_URL = %q", got)
	}
}

func TestActualNonDockerResourceManifestsResolveNativeExports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	skipIfRepoSecretsRequireMigrationOrKey(t, root)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home: %v", err)
	}

	openrouterEnv, err := LoadResourceEnvironment(root, home, "openrouter")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(openrouter): %v", err)
	}
	if got := openrouterEnv["RESOURCE_OPENROUTER_URL"]; got != "https://openrouter.ai/api/v1" {
		t.Fatalf("RESOURCE_OPENROUTER_URL = %q", got)
	}

	homeAssistantEnv, err := LoadResourceEnvironment(root, home, "home-assistant")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(home-assistant): %v", err)
	}
	if got := homeAssistantEnv["HOME_ASSISTANT_BASE_URL"]; got != "http://localhost:8123" {
		t.Fatalf("HOME_ASSISTANT_BASE_URL = %q", got)
	}

	whisperEnv, err := LoadResourceEnvironment(root, home, "whisper")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(whisper): %v", err)
	}
	if got := whisperEnv["WHISPER_URL"]; got != "http://localhost:8090" {
		t.Fatalf("WHISPER_URL = %q", got)
	}

	kokoroEnv, err := LoadResourceEnvironment(root, home, "kokoro")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(kokoro): %v", err)
	}
	if got := kokoroEnv["KOKORO_URL"]; got != "http://localhost:8880" {
		t.Fatalf("KOKORO_URL = %q", got)
	}

	judge0Env, err := LoadResourceEnvironment(root, home, "judge0")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(judge0): %v", err)
	}
	if got := judge0Env["JUDGE0_BASE_URL"]; got != "http://localhost:2358" {
		t.Fatalf("JUDGE0_BASE_URL = %q", got)
	}

	claudeEnv, err := LoadResourceEnvironment(root, home, "claude-code")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(claude-code): %v", err)
	}
	if got := claudeEnv["CLAUDE_CODE_PATH"]; got != "claude" {
		t.Fatalf("CLAUDE_CODE_PATH = %q", got)
	}

	codexEnv, err := LoadResourceEnvironment(root, home, "codex")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(codex): %v", err)
	}
	if got := codexEnv["CODEX_PATH"]; got != "codex" {
		t.Fatalf("CODEX_PATH = %q", got)
	}

	opencodeEnv, err := LoadResourceEnvironment(root, home, "opencode")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(opencode): %v", err)
	}
	if got := opencodeEnv["OPENCODE_XDG_CONFIG_HOME"]; got != filepath.Join(home, ".config", "vrooli", "resources", "opencode", "xdg-config") {
		t.Fatalf("OPENCODE_XDG_CONFIG_HOME = %q", got)
	}
}

func writePostgresManifestFixture(t *testing.T, root string) {
	t.Helper()
	writeEnvManifestFixture(t, root, "postgres", manifestpkg.ResourceManifest{
		Name:            "postgres",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "postgresql", Container: 5432, Host: 5433}},
		Runtime: manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
			Env: map[string]string{
				"POSTGRES_DB":   "vrooli",
				"POSTGRES_USER": "vrooli",
			},
		},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"POSTGRES_HOST": "localhost", "POSTGRES_SSLMODE": "disable"},
			FromPorts:      map[string]string{"POSTGRES_PORT": "postgresql"},
			FromRuntimeEnv: []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"POSTGRES_URL": {Template: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE}"},
				"DATABASE_URL": {Template: "${POSTGRES_URL}"},
			},
		},
	})
}

func skipIfRepoSecretsRequireMigrationOrKey(t *testing.T, root string) {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("resolve home dir: %v", err)
	}
	store := secrets.NewUserStore(home)
	if _, err := os.Stat(store.EncryptedPath()); err == nil {
		if strings.TrimSpace(os.Getenv(secrets.KeyEnvVar)) == "" {
			t.Skipf("user secrets at %s are encrypted but %s is unset", store.EncryptedPath(), secrets.KeyEnvVar)
		}
	}
}
