package securestore

import (
	"path/filepath"
	"testing"
)

func TestEnsureSetupBackendPersistsWritableNativeChoice(t *testing.T) {
	replaceSetupBackendSeams(t)
	dir := t.TempDir()
	backendSelectionPath = func() (string, error) { return filepath.Join(dir, "selection.json"), nil }
	diagnoseNativeForSetupFn = func() Diagnosis {
		return Diagnosis{Adapter: "test-native", Condition: "available", Writable: true}
	}

	status, err := EnsureSetupBackend()
	if err != nil {
		t.Fatalf("EnsureSetupBackend: %v", err)
	}
	if !status.Ready || status.SelectedBackend != BackendNative || !status.SelectionPersisted {
		t.Fatalf("unexpected setup status: %+v", status)
	}
	backend, found, err := SelectedBackend()
	if err != nil || !found || backend != BackendNative {
		t.Fatalf("persisted backend = %q, found=%t, err=%v", backend, found, err)
	}
}

func TestEnsureSetupBackendPersistsEncryptedChoiceWhenNativeIsNotWritable(t *testing.T) {
	replaceSetupBackendSeams(t)
	dir := t.TempDir()
	backendSelectionPath = func() (string, error) { return filepath.Join(dir, "selection.json"), nil }
	diagnoseNativeForSetupFn = func() Diagnosis {
		return Diagnosis{Adapter: "libsecret", Condition: "unavailable", Explanation: "the keyring session is locked"}
	}
	diagnoseSelectedForSetupFn = func() Diagnosis {
		return Diagnosis{Adapter: BackendEncryptedFile, Backend: BackendEncryptedFile, Condition: "absent", WriteCondition: "absent"}
	}

	status, err := EnsureSetupBackend()
	if err != nil {
		t.Fatalf("EnsureSetupBackend: %v", err)
	}
	if status.SelectedBackend != BackendEncryptedFile || status.Ready || !status.NeedsOperatorInput {
		t.Fatalf("unexpected setup status: %+v", status)
	}
	if status.OperatorAction == "" || status.Explanation == "" {
		t.Fatalf("setup status omitted remediation metadata: %+v", status)
	}
}

func TestEnsureSetupBackendHonorsPersistedChoice(t *testing.T) {
	replaceSetupBackendSeams(t)
	dir := t.TempDir()
	backendSelectionPath = func() (string, error) { return filepath.Join(dir, "selection.json"), nil }
	if err := SelectBackend(BackendEncryptedFile, "test policy"); err != nil {
		t.Fatalf("SelectBackend: %v", err)
	}
	diagnoseNativeForSetupFn = func() Diagnosis {
		t.Fatal("setup re-probed native backend after a persisted choice")
		return Diagnosis{}
	}
	diagnoseSelectedForSetupFn = func() Diagnosis {
		return Diagnosis{Adapter: BackendEncryptedFile, Backend: BackendEncryptedFile, Condition: "available", Available: true, Writable: true}
	}

	status, err := EnsureSetupBackend()
	if err != nil {
		t.Fatalf("EnsureSetupBackend: %v", err)
	}
	if !status.Ready || status.SelectedBackend != BackendEncryptedFile {
		t.Fatalf("unexpected setup status: %+v", status)
	}
}

func replaceSetupBackendSeams(t *testing.T) {
	t.Helper()
	previousPath := backendSelectionPath
	previousSelected := selectedBackendForSetupFn
	previousSelect := selectBackendForSetupFn
	previousNative := diagnoseNativeForSetupFn
	previousSelectedDiagnosis := diagnoseSelectedForSetupFn
	t.Cleanup(func() {
		backendSelectionPath = previousPath
		selectedBackendForSetupFn = previousSelected
		selectBackendForSetupFn = previousSelect
		diagnoseNativeForSetupFn = previousNative
		diagnoseSelectedForSetupFn = previousSelectedDiagnosis
	})
}
