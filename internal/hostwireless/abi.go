package hostwireless

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var nobleModules = []string{"mt76-connac-lib", "mt792x-lib", "mt7925-common", "mt7925e"}

// ABIProof records exact import and export checks for the selected modules.
type ABIProof struct {
	StockExports int `json:"stock_exports"`
	NewExports   int `json:"new_exports"`
	Imports      int `json:"imports"`
}

type symbol struct{ crc, module, scope, namespace string }

func readSymbols(path string) (map[string]symbol, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	symbols := map[string]symbol{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || len(fields) > 5 {
			return nil, fmt.Errorf("malformed symbol record in %s", path)
		}
		s := symbol{crc: fields[0], module: strings.ReplaceAll(filepath.Base(fields[2]), "_", "-"), scope: fields[3]}
		if len(fields) == 5 {
			s.namespace = fields[4]
		}
		if _, found := symbols[fields[1]]; found {
			return nil, fmt.Errorf("duplicate symbol %s in %s", fields[1], path)
		}
		symbols[fields[1]] = s
	}
	return symbols, nil
}

func verifyABI(stockPath, dir string, modules []string, newProviders map[string]string) (ABIProof, error) {
	var proof ABIProof
	stock, err := readSymbols(stockPath)
	if err != nil {
		return proof, err
	}
	built, err := readSymbols(filepath.Join(dir, "Module.symvers"))
	if err != nil {
		return proof, err
	}
	selected := map[string]bool{}
	for _, module := range modules {
		selected[module] = true
	}
	for name, original := range stock {
		if !selected[original.module] {
			continue
		}
		if built[name] != original {
			return proof, fmt.Errorf("existing export %s changed or is missing", name)
		}
		proof.StockExports++
	}
	if proof.StockExports == 0 {
		return proof, fmt.Errorf("no stock exports checked")
	}
	for name, current := range built {
		if !selected[current.module] {
			return proof, fmt.Errorf("unexpected export provider %s", current.module)
		}
		if _, exists := stock[name]; !exists {
			if newProviders[name] != current.module || current.scope != "EXPORT_SYMBOL_GPL" {
				return proof, fmt.Errorf("unexpected new export %s", name)
			}
			proof.NewExports++
		} else if stock[name] != current {
			return proof, fmt.Errorf("export %s conflicts with an existing provider", name)
		}
		stock[name] = current
	}
	if proof.NewExports != len(newProviders) {
		return proof, fmt.Errorf("missing expected new export")
	}
	for _, module := range modules {
		path := filepath.Join(dir, module+".modversions")
		data, err := os.ReadFile(path)
		if err != nil {
			return proof, err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 || stock[fields[1]].crc != fields[0] {
				return proof, fmt.Errorf("%s has an unresolved or incompatible import %q", module, line)
			}
			proof.Imports++
		}
	}
	return proof, nil
}
