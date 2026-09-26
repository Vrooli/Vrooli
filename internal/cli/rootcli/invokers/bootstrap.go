package invokers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// bootstrapFixture pins the argv scenarios/vrooli-bridge/bootstrap/bootstrap.sh
// builds; bootstrap_test.sh asserts the script really produces its first line.
const bootstrapFixture = "scenarios/vrooli-bridge/bootstrap/argv-fixture.txt"

// resultFilePlaceholder is the token the fixture uses for the mktemp path.
const resultFilePlaceholder = "<result-file>"

func bootstrapInvokers() ([]Invoker, error) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root for %s: %w", bootstrapFixture, err)
	}
	file, err := os.Open(filepath.Join(root, bootstrapFixture))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var items []Invoker
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		argv := strings.Fields(strings.ReplaceAll(text, resultFilePlaceholder, "/tmp/setup-result"))
		items = append(items, static(fmt.Sprintf("bridge-bootstrap/line-%d", line), bootstrapFixture, argv))
	}
	return items, scanner.Err()
}
