package facts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// BenchmarkLegacyProjectSearchCurrentCorpus is the pre-cutover comparison
// harness. Run it with -benchtime=1x: the legacy implementation scans the full
// repository and intentionally must not be repeated by Go's adaptive benchmark
// loop. CODE_FACTS_BENCH_REPO_ROOT selects a reproducible corpus snapshot.
func BenchmarkLegacyProjectSearchCurrentCorpus(b *testing.B) {
	root := os.Getenv("CODE_FACTS_BENCH_REPO_ROOT")
	if root == "" {
		root = repositoryRoot(b)
	}
	benchmarkLegacyProjectSearch(b, root)
}

// BenchmarkLegacyProjectSearchThreeTimesCorpus uses a caller-prepared
// deterministic three-times fixture. Requiring an explicit root prevents an
// ordinary package benchmark from copying the repository or consuming large
// amounts of storage without operator intent.
func BenchmarkLegacyProjectSearchThreeTimesCorpus(b *testing.B) {
	root := os.Getenv("CODE_FACTS_BENCH_3X_ROOT")
	if root == "" {
		b.Skip("set CODE_FACTS_BENCH_3X_ROOT to the prepared three-times corpus fixture")
	}
	benchmarkLegacyProjectSearch(b, root)
}

func benchmarkLegacyProjectSearch(b *testing.B, root string) {
	if b.N != 1 {
		b.Fatalf("legacy full-corpus benchmark requires -benchtime=1x; got b.N=%d", b.N)
	}
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		b.Fatalf("benchmark root %q is not a directory: %v", root, err)
	}
	target := &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, RepoRoot: filepath.Clean(root)}
	b.ReportAllocs()
	b.ResetTimer()
	response, err := lexicalProjectReport(context.Background(), root, target, []factsv1.FactFamily{
		factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
		factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
	}, "provider demotion", 5)
	b.StopTimer()
	if err != nil {
		b.Fatal(err)
	}
	if len(response.GetFacts()) == 0 {
		b.Fatal("legacy benchmark returned no authoritative result")
	}
}
