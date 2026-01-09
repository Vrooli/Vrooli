package deps

import (
	"testing"

	"scenario-to-desktop-runtime/manifest"
)

func TestTopoSort(t *testing.T) {
	t.Run("sorts services with no dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "a"},
			{ID: "b"},
			{ID: "c"},
		}

		order, err := TopoSort(services)
		if err != nil {
			t.Fatalf("TopoSort() error = %v", err)
		}

		if len(order) != 3 {
			t.Errorf("TopoSort() returned %d elements, want 3", len(order))
		}
	})

	t.Run("respects dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "api", Dependencies: []string{"db"}},
			{ID: "db"},
			{ID: "cache"},
		}

		order, err := TopoSort(services)
		if err != nil {
			t.Fatalf("TopoSort() error = %v", err)
		}

		// Find indices
		dbIdx, apiIdx := -1, -1
		for i, id := range order {
			if id == "db" {
				dbIdx = i
			}
			if id == "api" {
				apiIdx = i
			}
		}

		if dbIdx == -1 || apiIdx == -1 {
			t.Fatalf("TopoSort() missing expected services, got %v", order)
		}

		if dbIdx > apiIdx {
			t.Errorf("TopoSort() db should come before api, got order %v", order)
		}
	})

	t.Run("handles chain of dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "c", Dependencies: []string{"b"}},
			{ID: "b", Dependencies: []string{"a"}},
			{ID: "a"},
		}

		order, err := TopoSort(services)
		if err != nil {
			t.Fatalf("TopoSort() error = %v", err)
		}

		// a should come first, then b, then c
		if len(order) != 3 {
			t.Fatalf("TopoSort() returned %d elements, want 3", len(order))
		}

		idxA, idxB, idxC := -1, -1, -1
		for i, id := range order {
			switch id {
			case "a":
				idxA = i
			case "b":
				idxB = i
			case "c":
				idxC = i
			}
		}

		if idxA > idxB || idxB > idxC {
			t.Errorf("TopoSort() should order a < b < c, got %v", order)
		}
	})

	t.Run("detects cycle", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "a", Dependencies: []string{"b"}},
			{ID: "b", Dependencies: []string{"a"}},
		}

		_, err := TopoSort(services)
		if err == nil {
			t.Error("TopoSort() should detect cycle")
		}
	})

	t.Run("handles multiple dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "app", Dependencies: []string{"api", "cache"}},
			{ID: "api", Dependencies: []string{"db"}},
			{ID: "cache"},
			{ID: "db"},
		}

		order, err := TopoSort(services)
		if err != nil {
			t.Fatalf("TopoSort() error = %v", err)
		}

		if len(order) != 4 {
			t.Fatalf("TopoSort() returned %d elements, want 4", len(order))
		}

		// app should be last since it depends on api and cache
		idxApp := -1
		for i, id := range order {
			if id == "app" {
				idxApp = i
			}
		}

		if idxApp != len(order)-1 {
			t.Errorf("TopoSort() app should be last, got order %v", order)
		}
	})

	t.Run("handles empty services list", func(t *testing.T) {
		order, err := TopoSort([]manifest.Service{})
		if err != nil {
			t.Fatalf("TopoSort() error = %v", err)
		}
		if len(order) != 0 {
			t.Errorf("TopoSort([]) should return empty, got %v", order)
		}
	})
}

func TestFindService(t *testing.T) {
	services := []manifest.Service{
		{ID: "api", Type: "api"},
		{ID: "db", Type: "database"},
		{ID: "cache", Type: "cache"},
	}

	t.Run("finds existing service", func(t *testing.T) {
		svc := FindService(services, "db")
		if svc == nil {
			t.Fatal("FindService() returned nil for existing service")
		}
		if svc.ID != "db" {
			t.Errorf("FindService() returned wrong service, got ID=%q", svc.ID)
		}
		if svc.Type != "database" {
			t.Errorf("FindService() returned wrong type, got Type=%q", svc.Type)
		}
	})

	t.Run("returns nil for non-existent service", func(t *testing.T) {
		svc := FindService(services, "nonexistent")
		if svc != nil {
			t.Errorf("FindService() should return nil for non-existent, got %+v", svc)
		}
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		svc := FindService([]manifest.Service{}, "api")
		if svc != nil {
			t.Errorf("FindService() should return nil for empty slice, got %+v", svc)
		}
	})

	t.Run("returns pointer to original slice element", func(t *testing.T) {
		svc := FindService(services, "api")
		if svc == nil {
			t.Fatal("FindService() returned nil")
		}
		// Verify it's the same underlying data
		if svc != &services[0] {
			t.Error("FindService() should return pointer to original slice element")
		}
	})
}
