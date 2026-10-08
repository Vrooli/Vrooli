package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkExtractProfiles is intentionally small and deterministic enough to
// run locally while still exercising the real go/packages seam. Use it for
// relative comparisons between profiles, not as a machine-specific SLA.
func BenchmarkExtractProfiles(b *testing.B) {
	modulePath, err := filepath.Abs(filepath.Join("..", "..", "..", "bas", "fixtures", "go-usage-facts"))
	if err != nil {
		b.Fatalf("resolve fixture: %v", err)
	}
	profiles := []ExtractionProfile{
		ExtractionProfileFull,
		ExtractionProfileSemantic,
		ExtractionProfileStructural,
	}
	for _, profile := range profiles {
		b.Run(string(profile), func(b *testing.B) {
			service := NewService(NewPackagesLoader(), NewPathMutex())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, stats, err := service.ExtractWithStats(context.Background(), ExtractInput{
					ModulePath: modulePath,
					Profile:    profile,
				})
				if err != nil {
					b.Fatalf("extract %s: %v", profile, err)
				}
				b.ReportMetric(float64(stats.Load.Microseconds()), "load-us")
				b.ReportMetric(float64(stats.Normalize.Microseconds()), "normalize-us")
			}
		})
	}
}

func BenchmarkExtractCacheHit(b *testing.B) {
	modulePath, err := filepath.Abs(filepath.Join("..", "..", "..", "bas", "fixtures", "go-usage-facts"))
	if err != nil {
		b.Fatalf("resolve fixture: %v", err)
	}
	cache, err := NewFileExtractionCache(filepath.Join(b.TempDir(), "cache"))
	if err != nil {
		b.Fatalf("create cache: %v", err)
	}
	service := NewServiceWithCache(NewPackagesLoader(), NewPathMutex(), 1, cache)
	input := ExtractInput{ModulePath: modulePath, Profile: ExtractionProfileFull}
	if _, _, _, err := service.ExtractWithStats(context.Background(), input); err != nil {
		b.Fatalf("warm cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, stats, err := service.ExtractWithStats(context.Background(), input)
		if err != nil {
			b.Fatalf("cache hit: %v", err)
		}
		if !stats.CacheHit {
			b.Fatal("expected cache hit")
		}
		b.ReportMetric(float64(stats.Fingerprint.Microseconds()), "fingerprint-us")
	}
}

func BenchmarkExtractCacheHitLarge(b *testing.B) {
	if os.Getenv("GO_CODE_GRAPH_BENCH_LARGE") == "" {
		b.Skip("set GO_CODE_GRAPH_BENCH_LARGE=1 to benchmark the API module")
	}
	modulePath, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		b.Fatalf("resolve API module: %v", err)
	}
	cache, err := NewFileExtractionCache(filepath.Join(b.TempDir(), "cache"))
	if err != nil {
		b.Fatalf("create cache: %v", err)
	}
	service := NewServiceWithCache(NewPackagesLoader(), NewPathMutex(), 1, cache)
	input := ExtractInput{ModulePath: modulePath, Profile: ExtractionProfileStructural}
	if _, _, _, err := service.ExtractWithStats(context.Background(), input); err != nil {
		b.Fatalf("warm cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, stats, err := service.ExtractWithStats(context.Background(), input)
		if err != nil {
			b.Fatalf("cache hit: %v", err)
		}
		if !stats.CacheHit {
			b.Fatal("expected cache hit")
		}
		b.ReportMetric(float64(stats.Fingerprint.Microseconds()), "fingerprint-us")
	}
}
