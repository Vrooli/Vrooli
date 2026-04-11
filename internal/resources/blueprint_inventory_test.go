package resources

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBlueprintInventoryCoverageMatchesPhase0Plan(t *testing.T) {
	controller := NewController(projectRootForInventoryTest(t), t.TempDir())

	blueprints, err := controller.ListBlueprints()
	if err != nil {
		t.Fatalf("ListBlueprints: %v", err)
	}

	actual := make(map[string]struct{}, len(blueprints))
	for _, item := range blueprints {
		actual[item.Name] = struct{}{}
	}

	expected, err := loadPhase0BlueprintNames(projectRootForInventoryTest(t))
	if err != nil {
		t.Fatalf("loadPhase0BlueprintNames: %v", err)
	}

	missing := setDifference(expected, actual)
	unexpected := setDifference(actual, expected)

	if len(missing) > 0 || len(unexpected) > 0 {
		t.Fatalf("phase0 blueprint coverage drift detected\nmissing blueprint records: %v\nunexpected blueprint records: %v", missing, unexpected)
	}
}

func projectRootForInventoryTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}

func loadPhase0BlueprintNames(root string) (map[string]struct{}, error) {
	path := filepath.Join(root, "docs", "resources", "resource-phase0-inventory.md")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	items := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}
		resourceName := strings.TrimSpace(parts[1])
		proposedState := strings.TrimSpace(parts[6])
		if proposedState != "`blueprint`" {
			continue
		}
		resourceName = strings.Trim(resourceName, "`")
		if resourceName != "" {
			items[resourceName] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func setDifference(left, right map[string]struct{}) []string {
	items := make([]string, 0)
	for key := range left {
		if _, ok := right[key]; ok {
			continue
		}
		items = append(items, key)
	}
	sort.Strings(items)
	return items
}
