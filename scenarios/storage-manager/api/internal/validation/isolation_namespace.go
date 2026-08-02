package validation

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	register(&isoNamespace{})
}

// isoNamespace emits STORAGE_NAMESPACE_HARDCODED when a Go file hardcodes a
// Qdrant collection name or Redis key prefix (embedding the scenario slug)
// instead of resolving it through the variant-aware api-core storage helpers
// (storage.Collection / storage.RedisKey / storage.RedisPrefix). A hardcoded
// namespace has no variant dimension, so a Baseline-Modes shadow instance reads
// and writes the LIVE keyspace/collection — corrupting the very state a shadow
// engagement exists to protect.
//
// Ported verbatim in spirit from scenario-auditor's storage_namespace_helpers
// rule (high-precision: favours false negatives). A file that calls the helper
// for an engine is treated as adopted for that engine.
type isoNamespace struct{}

func (isoNamespace) Name() string { return "isolation.namespace-hardcoded" }

func (isoNamespace) Applies(ac AnalyzerContext) bool {
	// Only meaningful for Go scenarios that touch a namespaced engine.
	return ac.IsGo() && (ac.HasEngine(EngineQdrant) || ac.HasEngine(EngineRedis))
}

func (isoNamespace) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, isoScanNamespaceFile(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

var (
	isoQdrantEmbeddingsLiteral = regexp.MustCompile(`"[a-z0-9][a-z0-9_-]*[_-]embeddings"`)
	isoCollectionConstLiteral  = regexp.MustCompile(`(?i)\b[a-z0-9_]*collection[a-z0-9_]*\s*(?::=|=)\s*"[^"]+"`)
	isoCollectionFieldLiteral  = regexp.MustCompile(`(?i)\bcollection(?:name)?\s*:\s*"[^"]+"`)
	isoRedisNamespaceConst     = regexp.MustCompile(`(?i)\b[a-z0-9_]*(?:keyprefix|rediskey|keyspace|namespace|prefix)[a-z0-9_]*\s*(?::=|=)\s*"[^"]+"`)
	isoGoStringLiteral         = regexp.MustCompile(`"[^"]*"`)
	isoQdrantContext           = regexp.MustCompile(`(?i)qdrant|aisearch|embedding|\bvector`)
	isoRedisContext            = regexp.MustCompile(`(?i)\bredis\b|go-redis|\brdb\b|redisclient`)
)

// isoScanNamespaceFile is the per-file detector. It mirrors the auditor rule's
// high-precision gating: a finding fires only when the file shows real
// Redis/Qdrant context AND does not already route that engine through the
// helper.
func isoScanNamespaceFile(content, relPath string) []Finding {
	if !strings.HasSuffix(strings.ToLower(relPath), ".go") {
		return nil
	}
	usesCollectionHelper := strings.Contains(content, "storage.Collection(")
	usesRedisHelper := strings.Contains(content, "storage.RedisKey(") ||
		strings.Contains(content, "storage.RedisPrefix(")
	hasQdrantContext := isoQdrantContext.MatchString(content)
	hasRedisContext := isoRedisContext.MatchString(content)

	if usesCollectionHelper && usesRedisHelper {
		return nil
	}
	if !hasQdrantContext && !hasRedisContext {
		return nil
	}

	var findings []Finding
	seenQdrant := map[int]bool{}
	seenRedis := map[int]bool{}
	for i, raw := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		if !usesCollectionHelper && !seenQdrant[i] {
			hit := isoQdrantEmbeddingsLiteral.MatchString(raw)
			if !hit && hasQdrantContext {
				hit = isoCollectionConstLiteral.MatchString(raw) || isoCollectionFieldLiteral.MatchString(raw)
			}
			if hit {
				seenQdrant[i] = true
				findings = append(findings, Finding{
					Code:        "STORAGE_NAMESPACE_HARDCODED",
					Severity:    SeverityWarning,
					Title:       "Hardcoded Qdrant collection namespace",
					Message:     "Hardcoded Qdrant collection name embeds the scenario slug instead of resolving it through storage.Collection(domain); a Baseline-Modes shadow instance would read and write the LIVE collection, corrupting protected state.",
					Location:    isoLoc(relPath, i+1),
					Remediation: "Resolve the collection at runtime via storage.Collection(\"<domain>\") (packages/api-core/storage) so live and shadow address different collections. See storage-steer §4.2/§4.5.",
					Analyzer:    "isolation.namespace-hardcoded",
				})
			}
		}

		if !usesRedisHelper && hasRedisContext && !seenRedis[i] {
			hit := isoRedisNamespaceConst.MatchString(raw)
			if !hit {
				for _, lit := range isoGoStringLiteral.FindAllString(raw, -1) {
					if isoRedisKeyShape(lit) {
						hit = true
						break
					}
				}
			}
			if hit {
				seenRedis[i] = true
				findings = append(findings, Finding{
					Code:        "STORAGE_NAMESPACE_HARDCODED",
					Severity:    SeverityWarning,
					Title:       "Hardcoded Redis key namespace",
					Message:     "Hardcoded Redis key/prefix embeds the scenario slug and has no variant dimension; a Baseline-Modes shadow instance would write into the LIVE keyspace.",
					Location:    isoLoc(relPath, i+1),
					Remediation: "Build keys through storage.RedisKey(\"<domain>\", segments...) or storage.RedisPrefix(\"<domain>\") (packages/api-core/storage). See storage-steer §4.3/§4.5.",
					Analyzer:    "isolation.namespace-hardcoded",
				})
			}
		}
	}
	return findings
}

// isoRedisKeyShape reports whether a quoted literal looks like a
// scenario-prefixed Redis key or prefix, rejecting URLs, host:port, and
// timestamps. Ported from the auditor rule.
func isoRedisKeyShape(lit string) bool {
	s := strings.TrimSuffix(strings.TrimPrefix(lit, `"`), `"`)
	if s == "" || !strings.Contains(s, ":") {
		return false
	}
	if strings.Contains(s, "//") || strings.Contains(s, "@") || strings.Contains(s, " ") {
		return false
	}
	first := s
	if idx := strings.Index(s, ":"); idx >= 0 {
		first = s[:idx]
	}
	if first == "" || first[0] < 'a' || first[0] > 'z' {
		return false
	}
	if strings.Count(s, ":") >= 2 {
		return true
	}
	return strings.HasSuffix(s, ":") || strings.HasSuffix(s, ":*")
}

func isoLoc(relPath string, line int) string {
	return relPath + ":" + strconv.Itoa(line)
}
