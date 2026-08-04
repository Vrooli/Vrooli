package securestore

import "fmt"

// BackendSetupStatus is the metadata-safe result of setup's authority choice.
// It never contains credential values or passphrases.
type BackendSetupStatus struct {
	SelectedBackend    string
	SelectionPersisted bool
	NativeAdapter      string
	NativeCondition    string
	NativeWritable     bool
	Ready              bool
	NeedsOperatorInput bool
	Explanation        string
	OperatorAction     string
}

var (
	selectedBackendForSetupFn  = SelectedBackend
	selectBackendForSetupFn    = SelectBackend
	diagnoseNativeForSetupFn   = DiagnoseNativeWritable
	diagnoseSelectedForSetupFn = DiagnoseWritable
)

// EnsureSetupBackend chooses the credential authority once for this
// installation. A native backend must pass a real write/read/delete probe to
// be selected. Otherwise the encrypted Vrooli backend is selected, including
// on headless hosts and on hosts whose desktop keyring is present but locked.
//
// The function deliberately does not fall back per operation. A transient
// native outage after this decision leaves the selected backend unchanged.
func EnsureSetupBackend() (BackendSetupStatus, error) {
	if backend, found, err := selectedBackendForSetupFn(); err != nil {
		return BackendSetupStatus{}, err
	} else if found {
		return inspectSelectedBackend(backend)
	}

	native := diagnoseNativeForSetupFn()
	if native.Writable {
		if err := selectBackendForSetupFn(BackendNative, "native backend passed setup write/read/delete readiness"); err != nil {
			return BackendSetupStatus{}, err
		}
		return BackendSetupStatus{
			SelectedBackend:    BackendNative,
			SelectionPersisted: true,
			NativeAdapter:      native.Adapter,
			NativeCondition:    native.Condition,
			NativeWritable:     true,
			Ready:              true,
		}, nil
	}

	if err := selectBackendForSetupFn(BackendEncryptedFile, "native backend did not pass setup write/read/delete readiness"); err != nil {
		return BackendSetupStatus{}, err
	}
	status, err := inspectSelectedBackend(BackendEncryptedFile)
	if err != nil {
		return BackendSetupStatus{}, err
	}
	status.NativeAdapter = native.Adapter
	status.NativeCondition = native.Condition
	status.NativeWritable = false
	status.Explanation = native.Explanation
	if status.Explanation == "" {
		status.Explanation = "native credential storage was not writable during setup"
	}
	return status, nil
}

func inspectSelectedBackend(backend string) (BackendSetupStatus, error) {
	status := BackendSetupStatus{SelectedBackend: backend, SelectionPersisted: true}
	diagnosis := diagnoseSelectedForSetupFn()
	status.Ready = diagnosis.Available && diagnosis.Writable
	status.Explanation = diagnosis.Explanation
	status.OperatorAction = diagnosis.WriteFix
	if backend == BackendEncryptedFile && diagnosis.Condition == "absent" {
		status.NeedsOperatorInput = true
		status.OperatorAction = "complete encrypted credential-store initialization through Vrooli onboarding"
	}
	if !status.Ready && status.Explanation == "" {
		status.Explanation = fmt.Sprintf("selected credential backend %q is not ready", backend)
	}
	return status, nil
}
