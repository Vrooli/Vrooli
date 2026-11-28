package scenarioruntime

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// schemaHistory holds the canonical DDL of every format this binary can
// decode. The write version's snapshot must equal the live schema, which keeps
// the history honest; older snapshots are frozen so a later schema edit cannot
// silently change the meaning of a version both images already know.
//
//go:embed schema_history/*.sql
var schemaHistoryFS embed.FS

// additiveFromRuntimeRegistry is the oldest format whose reader can consume
// everything this binary writes. It is pinned, never derived: the claim is
// earned by TestRuntimeRegistryAdditiveFromFixture, and a bump to a new
// SchemaVersion must re-earn it or reset it to zero.
const additiveFromRuntimeRegistry = 16

var (
	schemaHistoryOnce sync.Once
	schemaHistory     map[int]string
	schemaHistoryErr  error
)

// registryStatePath is the registry's durable file, relative to the runtime
// home. It must match the repo contract's runtime_home/runtime_db entry, which
// TestRegistryStatePathMatchesRepoContract enforces.
const registryStatePath = "state/runtime.db"

// StateFormat describes the runtime registry as a versioned persistent store.
//
// Reads is exactly the set of formats with a frozen schema snapshot. Older
// databases are not "read" in that sense; they are migrated forward, which is
// what MigratesFrom claims.
func StateFormat() stateformat.Store {
	schemas := registrySchemaDigests()
	reads := make([]stateformat.Version, 0, len(schemas))
	byVersion := make(map[stateformat.Version]string, len(schemas))
	for version, digest := range schemas {
		reads = append(reads, stateformat.Version(version))
		byVersion[stateformat.Version(version)] = digest
	}
	sort.Slice(reads, func(i, j int) bool { return reads[i] < reads[j] })
	return stateformat.Store{
		Name:         "runtime-registry",
		Write:        stateformat.Version(SchemaVersion),
		Reads:        reads,
		MigratesFrom: stateformat.Version(SchemaVersion - 1),
		AdditiveFrom: stateformat.Version(additiveFromRuntimeRegistry),
		Schemas:      byVersion,
		StatePaths:   []string{registryStatePath},
	}
}

// registrySchemaDigests maps each frozen format to its canonical DDL digest.
func registrySchemaDigests() map[int]string {
	schemaHistoryOnce.Do(func() {
		entries, err := fs.ReadDir(schemaHistoryFS, "schema_history")
		if err != nil {
			schemaHistoryErr = err
			return
		}
		digests := make(map[int]string, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || path.Ext(entry.Name()) != ".sql" {
				continue
			}
			version, err := parseSchemaSnapshotName(entry.Name())
			if err != nil {
				schemaHistoryErr = err
				return
			}
			data, err := schemaHistoryFS.ReadFile(path.Join("schema_history", entry.Name()))
			if err != nil {
				schemaHistoryErr = err
				return
			}
			digests[version] = digestSchemaText(string(data))
		}
		schemaHistory = digests
	})
	if schemaHistoryErr != nil {
		panic(fmt.Sprintf("runtime registry schema history: %v", schemaHistoryErr))
	}
	return schemaHistory
}

func parseSchemaSnapshotName(name string) (int, error) {
	base := strings.TrimSuffix(name, ".sql")
	if !strings.HasPrefix(base, "v") {
		return 0, fmt.Errorf("snapshot %q must be named v<version>.sql", name)
	}
	version, err := strconv.Atoi(strings.TrimPrefix(base, "v"))
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("snapshot %q must be named v<version>.sql", name)
	}
	return version, nil
}

func digestSchemaText(text string) string {
	sum := sha256.Sum256([]byte(canonicalDDL(text)))
	return hex.EncodeToString(sum[:])
}

// canonicalDDL removes layout differences so a reformatted statement keeps its
// digest while a changed declaration does not.
func canonicalDDL(ddl string) string {
	return strings.Join(strings.Fields(ddl), " ")
}

// liveSchemaDigest is the canonical digest of the schema this binary creates.
func liveSchemaDigest() string { return digestSchemaText(schemaSQL) }
