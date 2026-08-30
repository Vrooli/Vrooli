package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func printDiff(path string, current matrix) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var baseline matrix
	if err := json.Unmarshal(data, &baseline); err != nil {
		return fmt.Errorf("parse baseline: %w", err)
	}
	left := make(map[string]matrixCell, len(baseline.Cells))
	for _, cell := range baseline.Cells {
		left[cell.AssetID+"\x00"+cell.Gate] = cell
	}
	right := make(map[string]matrixCell, len(current.Cells))
	for _, cell := range current.Cells {
		right[cell.AssetID+"\x00"+cell.Gate] = cell
	}
	keys := make(map[string]bool, len(left)+len(right))
	for key := range left {
		keys[key] = true
	}
	for key := range right {
		keys[key] = true
	}
	var moved []string
	for key := range keys {
		before, beforeOK := left[key]
		after, afterOK := right[key]
		if !beforeOK || !afterOK || before.Verdict != after.Verdict || !equalStrings(before.FindingCodes, after.FindingCodes) {
			moved = append(moved, fmt.Sprintf("%s: %+v -> %+v", key, before, after))
		}
	}
	for _, change := range moved {
		fmt.Println(change)
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
