package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

func main() {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fail(err)
	}
	inventory, err := monetization.BuildMeterInventory(root)
	if err != nil {
		fail(err)
	}
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		fail(err)
	}
	data = append(data, '\n')
	path := filepath.Join(root, "packages", "monetization-go", "meter-inventory.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fail(err)
	}
	fmt.Println(path)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
