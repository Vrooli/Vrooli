package api

import (
	"regexp"
	"strings"
)

/*
Rule: Variant-Aware Storage Namespaces
Description: Redis key prefixes and Qdrant collection names must be composed through the variant-aware api-core storage helpers (storage.Collection / storage.RedisKey / storage.RedisPrefix), never hardcoded as scenario-slug constants.
Reason: A hardcoded namespace has no variant dimension, so a Baseline-Modes shadow instance reads and writes the LIVE keyspace/collection — corrupting the very state the engagement exists to protect. Hardcoding the slug routes the scenario to live-only mode in the Baseline-Modes decision tree (it cannot be safely shadowed) and is a maturity-ladder regression.
Category: api
Severity: medium
Standard: storage-namespace-v1
Targets: api

This is the detection half of plan P5 (Variant-Aware Storage Namespace SSOT).
It feeds two consumers: the EM/storage-steer maturity loop (so the long tail of
scenarios migrates to the helpers) and the Baseline-Modes decision tree's
namespaceability gate (so a scenario that still hardcodes a Redis/Qdrant
namespace it WRITES is routed to live mode until it adopts the helpers).

Detection is deliberately high-precision (favours false negatives over false
positives): a finding fires only when the file shows real Redis/Qdrant context
AND does not already route the corresponding engine through the helper. A file
that calls storage.Collection / storage.RedisKey / storage.RedisPrefix is
treated as adopted for that engine. See storage-steer §4.2/§4.3/§4.5.

<test-case id="qdrant-embeddings-const" should-fail="true" path="api/internal/notes/qdrant.go">
  <description>❌ Hardcoded Qdrant collection const embedding the scenario slug</description>
  <input language="go"><![CDATA[
package notes

import "github.com/vrooli/ai-go/search"

const NotesCollection = "swarm-manager_notes_embeddings"

func ensure(c *aisearch.Client) error {
    return c.EnsureCollection(NotesCollection)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Qdrant</expected-message>
</test-case>

<test-case id="qdrant-collection-field" should-fail="true" path="api/internal/backlog/qdrant.go">
  <description>❌ Hardcoded Qdrant collection passed via a struct field</description>
  <input language="go"><![CDATA[
package backlog

func newStore(client *qdrant.Client) *Store {
    return &Store{Collection: "swarm-manager-backlog"}
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Qdrant</expected-message>
</test-case>

<test-case id="qdrant-uses-helper" should-fail="false" path="api/internal/notes/qdrant.go">
  <description>✅ Resolves the collection through storage.Collection at runtime</description>
  <input language="go"><![CDATA[
package notes

import "github.com/vrooli/api-core/storage"

func collectionName() (string, error) {
    return storage.Collection("notes")
}
]]></input>
</test-case>

<test-case id="redis-prefix-const" should-fail="true" path="api/internal/auth/redis.go">
  <description>❌ Hardcoded Redis key prefix const with no variant dimension</description>
  <input language="go"><![CDATA[
package auth

import "github.com/redis/go-redis/v9"

const sessionPrefix = "lpbs:auth:session:"

func sessionKey(rdb *redis.Client, id string) string {
    return sessionPrefix + id
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Redis</expected-message>
</test-case>

<test-case id="redis-sprintf-key" should-fail="true" path="api/internal/idea/redis.go">
  <description>❌ Hardcoded mid-string Redis key built with fmt.Sprintf</description>
  <input language="go"><![CDATA[
package idea

import (
    "fmt"

    "github.com/redis/go-redis/v9"
)

func researchKey(rdb *redis.Client, id string) string {
    return fmt.Sprintf("swarm-manager:idea:%s:research", id)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Redis</expected-message>
</test-case>

<test-case id="redis-uses-helper" should-fail="false" path="api/internal/auth/redis.go">
  <description>✅ Builds keys through storage.RedisKey</description>
  <input language="go"><![CDATA[
package auth

import "github.com/vrooli/api-core/storage"

func sessionKey(id string) (string, error) {
    return storage.RedisKey("auth", "session", id)
}
]]></input>
</test-case>

<test-case id="redis-host-port-not-flagged" should-fail="false" path="api/internal/cache/redis.go">
  <description>✅ A redis host:port address is not a key namespace</description>
  <input language="go"><![CDATA[
package cache

import "github.com/redis/go-redis/v9"

func client() *redis.Client {
    return redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}
]]></input>
</test-case>

<test-case id="non-storage-file-clean" should-fail="false" path="api/internal/web/handlers.go">
  <description>✅ A non-storage file with URLs and timestamps is untouched</description>
  <input language="go"><![CDATA[
package web

func config() {
    apiURL := "https://api.example.com/v1"
    layout := "15:04:05"
    _ = apiURL
    _ = layout
}
]]></input>
</test-case>
*/

var (
	// The dominant hardcoded Qdrant collection pattern: a string literal ending
	// in an embeddings suffix (e.g. "workflow_embeddings",
	// "swarm-manager_notes_embeddings").
	qdrantEmbeddingsLiteral = regexp.MustCompile(`"[a-z0-9][a-z0-9_-]*[_-]embeddings"`)

	// A const/var whose identifier names it a collection, assigned a string
	// literal (e.g. `const NotesCollection = "..."`). Gated on Qdrant context.
	collectionConstLiteral = regexp.MustCompile(`(?i)\b[a-z0-9_]*collection[a-z0-9_]*\s*(?::=|=)\s*"[^"]+"`)

	// A collection passed via a struct field (e.g. `Collection: "..."` /
	// `CollectionName: "..."`). Gated on Qdrant context.
	collectionFieldLiteral = regexp.MustCompile(`(?i)\bcollection(?:name)?\s*:\s*"[^"]+"`)

	// A const/var whose identifier names it a Redis prefix/key/namespace,
	// assigned a string literal. Gated on Redis context.
	redisNamespaceConstLiteral = regexp.MustCompile(`(?i)\b[a-z0-9_]*(?:keyprefix|rediskey|keyspace|namespace|prefix)[a-z0-9_]*\s*(?::=|=)\s*"[^"]+"`)

	// Any double-quoted string literal on a line (used to test redis-key shape).
	goStringLiteral = regexp.MustCompile(`"[^"]*"`)

	// Whole-file context predicates.
	qdrantContextPattern = regexp.MustCompile(`(?i)qdrant|aisearch|embedding|\bvector`)
	redisContextPattern  = regexp.MustCompile(`(?i)\bredis\b|go-redis|\brdb\b|redisclient`)
)

// CheckStorageNamespaceHelpers flags Redis/Qdrant namespaces that are hardcoded
// with the scenario slug instead of being resolved through the variant-aware
// api-core storage helpers.
func CheckStorageNamespaceHelpers(content []byte, filePath string) []Violation {
	if !strings.HasSuffix(strings.ToLower(filePath), ".go") {
		return nil
	}
	// api-core owns the helpers; never flag the SSOT itself.
	if isExemptPath(filePath) || isAPICorePath(filePath) {
		return nil
	}

	contentStr := string(content)

	// A file that calls the helper for an engine is considered adopted for that
	// engine — the SSOT routes its namespaces, so there is nothing to flag.
	usesCollectionHelper := strings.Contains(contentStr, "storage.Collection(")
	usesRedisHelper := strings.Contains(contentStr, "storage.RedisKey(") ||
		strings.Contains(contentStr, "storage.RedisPrefix(")

	hasQdrantContext := qdrantContextPattern.MatchString(contentStr)
	hasRedisContext := redisContextPattern.MatchString(contentStr)

	if usesCollectionHelper && usesRedisHelper {
		return nil
	}
	if !hasQdrantContext && !hasRedisContext {
		return nil
	}

	var violations []Violation
	seenQdrant := map[int]bool{}
	seenRedis := map[int]bool{}

	lines := strings.Split(contentStr, "\n")
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// --- Qdrant collection namespace ---
		if !usesCollectionHelper && !seenQdrant[i] {
			hit := qdrantEmbeddingsLiteral.MatchString(raw)
			if !hit && hasQdrantContext {
				hit = collectionConstLiteral.MatchString(raw) || collectionFieldLiteral.MatchString(raw)
			}
			if hit {
				seenQdrant[i] = true
				violations = append(violations, Violation{
					Type:           "hardcoded_qdrant_collection",
					Severity:       "medium",
					Title:          "Hardcoded Qdrant Collection Namespace",
					Description:    "Hardcoded Qdrant collection name embeds the scenario slug instead of resolving it through storage.Collection(domain); a Baseline-Modes shadow instance would read and write the live collection.",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    trimmed,
					Recommendation: "Resolve the collection at runtime via storage.Collection(\"<domain>\") (packages/api-core/storage) so live and shadow address different collections. See storage-steer §4.2/§4.5.",
					Standard:       "storage-namespace-v1",
				})
			}
		}

		// --- Redis key namespace ---
		if !usesRedisHelper && hasRedisContext && !seenRedis[i] {
			hit := redisNamespaceConstLiteral.MatchString(raw)
			if !hit {
				for _, lit := range goStringLiteral.FindAllString(raw, -1) {
					if isRedisKeyShape(lit) {
						hit = true
						break
					}
				}
			}
			if hit {
				seenRedis[i] = true
				violations = append(violations, Violation{
					Type:           "hardcoded_redis_namespace",
					Severity:       "medium",
					Title:          "Hardcoded Redis Key Namespace",
					Description:    "Hardcoded Redis key/prefix embeds the scenario slug and has no variant dimension; a Baseline-Modes shadow instance would write into the live keyspace.",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    trimmed,
					Recommendation: "Build keys through storage.RedisKey(\"<domain>\", segments...) or storage.RedisPrefix(\"<domain>\") (packages/api-core/storage). See storage-steer §4.3/§4.5.",
					Standard:       "storage-namespace-v1",
				})
			}
		}
	}

	return violations
}

// isRedisKeyShape reports whether a quoted string literal looks like a
// scenario-prefixed Redis key or key prefix (e.g. "scenario:domain:%s:research"
// or "scenario:domain:"), while rejecting URLs, host:port addresses, and
// timestamps that also contain colons.
func isRedisKeyShape(lit string) bool {
	s := strings.TrimSuffix(strings.TrimPrefix(lit, `"`), `"`)
	if s == "" || !strings.Contains(s, ":") {
		return false
	}
	// URLs and host strings are not key namespaces.
	if strings.Contains(s, "//") || strings.Contains(s, "@") || strings.Contains(s, " ") {
		return false
	}
	// The first segment must be a slug-like token (starts with a lowercase
	// letter), which excludes "15:04:05" and other non-namespace literals.
	first := s
	if idx := strings.Index(s, ":"); idx >= 0 {
		first = s[:idx]
	}
	if first == "" || first[0] < 'a' || first[0] > 'z' {
		return false
	}
	colons := strings.Count(s, ":")
	if colons >= 2 {
		return true
	}
	// A single colon only qualifies as a terminated prefix or a SCAN pattern,
	// which excludes host:port literals like "localhost:6379".
	return strings.HasSuffix(s, ":") || strings.HasSuffix(s, ":*")
}
