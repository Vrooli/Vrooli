package main

import (
	"os"
	"testing"
)

func TestREADMEHasCurrentGeneratedEnvironmentTable(t *testing.T) {
	document, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := replaceGeneratedBlock(string(document))
	if err != nil {
		t.Fatal(err)
	}
	if string(document) != updated {
		t.Fatal("README scenario environment table differs from cliapp.StandardScenarioEnv")
	}
}
