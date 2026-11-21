package executioninventory

import (
	"testing"

	"github.com/google/uuid"
)

func TestOwnedArtifactPathsAreScopedToKnownRootsAndExecution(t *testing.T) {
	executionID := uuid.NewString()
	if !IsOwnedPath("artifacts/screenshots/step.png", executionID) {
		t.Fatal("known screenshot path was rejected")
	}
	if !IsOwnedPath("artifacts/"+executionID+"/trace-1.zip", executionID) {
		t.Fatal("generic local artifact path for the exact execution was rejected")
	}
	if IsOwnedPath("artifacts/"+uuid.NewString()+"/trace-1.zip", executionID) {
		t.Fatal("artifact path belonging to another execution was accepted")
	}
	if IsOwnedPath("artifacts/unknown/step.bin", executionID) {
		t.Fatal("unknown artifact subtree was accepted")
	}
}

func TestOwnedDirectoriesAreScopedToKnownRootsAndExecution(t *testing.T) {
	executionID := uuid.NewString()
	if !IsOwnedDirectory("artifacts/screenshots", executionID) {
		t.Fatal("known screenshot directory was rejected")
	}
	if !IsOwnedDirectory("artifacts/"+executionID, executionID) {
		t.Fatal("generic local artifact directory for the exact execution was rejected")
	}
	if IsOwnedDirectory("artifacts/"+uuid.NewString(), executionID) {
		t.Fatal("artifact directory belonging to another execution was accepted")
	}
	if IsOwnedDirectory("artifacts/unknown", executionID) {
		t.Fatal("unknown artifact directory was accepted")
	}
}

func TestExecutionObjectKeysRequireCanonicalExecutionScope(t *testing.T) {
	executionID := uuid.NewString()
	valid := []string{
		"artifacts/" + executionID + "/trace.zip",
		executionID + "/artifacts/screenshots/step.png",
		executionID + "/artifacts/network/events.json",
	}
	for _, key := range valid {
		if !IsExecutionObjectKey(key, executionID) {
			t.Errorf("valid execution-scoped object key rejected: %q", key)
		}
	}
	invalid := []string{
		"artifacts/" + uuid.NewString() + "/trace.zip",
		executionID + "/artifacts/unknown/file.bin",
		executionID + "/artifacts/screenshots/../foreign.png",
		"/" + executionID + "/artifacts/screenshots/step.png",
	}
	for _, key := range invalid {
		if IsExecutionObjectKey(key, executionID) {
			t.Errorf("unscoped object key accepted: %q", key)
		}
	}
}
