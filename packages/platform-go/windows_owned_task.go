package platform

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	windowsOwnedTaskPrefix            = "VrooliOwned-"
	windowsOwnedRunIssuedValue        = "RunIssued"
	windowsOwnedRevokedValue          = "Revoked"
	windowsOwnedMarker         uint32 = 1
)

func windowsOwnedTaskName(i OperationIntent) string { return windowsOwnedTaskPrefix + i.InvocationID }

// Only the maintained early capsule reader performs task admission before any
// vendor or owner IPC. A generic executable cannot satisfy this protocol.
func validateWindowsMaintainedLaunchShape(i OperationIntent, s OperationSpec) error {
	if i.Owner != "resource-initial" && i.Owner != "resource-update" {
		return ErrUnsupported
	}
	if len(s.Args) != 2 || s.Args[0] != "__vrooli-openrgb-native-capsule" || s.Dir != filepath.Dir(s.Path) {
		return ErrUnsupported
	}
	path := s.Args[1]
	if len(path) > 4096 || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n%$") || filepath.Base(path) != "capsule.json" || filepath.Base(filepath.Dir(path)) != i.AttemptID || filepath.Base(filepath.Dir(filepath.Dir(path))) != "launch" || !filepath.IsAbs(s.Path) || filepath.Clean(s.Path) != s.Path || strings.ContainsAny(s.Path, "%$") {
		return ErrOperationIdentity
	}
	return nil
}

func windowsOwnedTaskXML(p windowsOwnedPayload) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><Task version="1.4" xmlns="%s"><Triggers/><Principals><Principal id="Operator"><UserId>%s</UserId><LogonType>S4U</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals><Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries><StopIfGoingOnBatteries>false</StopIfGoingOnBatteries><AllowHardTerminate>true</AllowHardTerminate><StartWhenAvailable>false</StartWhenAvailable><AllowStartOnDemand>true</AllowStartOnDemand><Enabled>true</Enabled><RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable><RunOnlyIfIdle>false</RunOnlyIfIdle><ExecutionTimeLimit>PT0S</ExecutionTimeLimit></Settings><Actions Context="Operator"><Exec><Command>%s</Command><Arguments>%s</Arguments><WorkingDirectory>%s</WorkingDirectory></Exec></Actions></Task>`, windowsTaskNamespace, xmlText(strings.TrimPrefix(p.Spec.NativeAccount, "sid:")), xmlText(p.Spec.Path), xmlText(windowsArguments(p.Spec.Args)), xmlText(p.Spec.Dir))
}

func windowsOwnedTaskPolicy(d windowsTaskDefinition) bool {
	return d.Settings.HardTerminate == "true" && len(d.Boot) == 0 && len(d.Time) == 0 && d.Settings.Limit == "PT0S" && d.Settings.Multiple == "IgnoreNew" && d.Settings.NoBatteryStart == "false" && d.Settings.StopOnBattery == "false" && d.Settings.Demand == "true" && d.Settings.StartAvailable == "false" && d.Settings.Network == "false" && d.Settings.Idle == "false" && d.Settings.RemoteApp == "false" && d.Settings.Volatile == "false" && d.Settings.Restart.Interval == "" && d.Settings.Restart.Count == ""
}

func windowsOwnedTaskDefinitionMatches(p windowsOwnedPayload, s WindowsTaskSnapshot) error {
	if s.Name != windowsOwnedTaskName(p.Intent) {
		return ErrOperationIdentity
	}
	d, e := parseWindowsTaskDefinition(s.XML)
	if e != nil {
		return e
	}
	expected, e := parseWindowsTaskDefinition(windowsOwnedTaskXML(p))
	if e != nil {
		return e
	}
	if !windowsOwnedTaskPolicy(d) || d.Principals[0] != expected.Principals[0] || d.Actions[0] != expected.Actions[0] {
		return ErrOperationIdentity
	}
	return nil
}

// These effects are the actual admission boundary, also exercised by fixtures.
// Reserving the protected intent is irreversible for replay purposes: any later
// failure requires lookup/abort, never a second native Run.
type windowsOwnedTaskSubmissionEffects struct {
	Reserve    func(windowsOwnedPayload) error
	Register   func(windowsOwnedPayload) error
	MarkIssued func(OperationIntent) error
	Run        func(string) error
}

func submitWindowsOwnedTask(p windowsOwnedPayload, e windowsOwnedTaskSubmissionEffects) error {
	if err := validateWindowsMaintainedLaunchShape(p.Intent, p.Spec); err != nil {
		return err
	}
	if e.Reserve == nil || e.Register == nil || e.MarkIssued == nil || e.Run == nil {
		return ErrOperationIdentity
	}
	if err := e.Reserve(p); err != nil {
		return err
	}
	if err := e.Register(p); err != nil {
		return err
	}
	if err := e.MarkIssued(p.Intent); err != nil {
		return err
	}
	return e.Run(windowsOwnedTaskName(p.Intent))
}

// Old boot process/instance identities cannot authorize signaling after reboot.
// Only independently absent native Job namespace permits old task retirement;
// a newly occupied same UUID name must be preserved as ambiguous/foreign.
func windowsOwnedRetirementTaskRoot(payloadBoot, currentBoot string, root windowsOwnedRootRecord, jobAbsent bool) (windowsOwnedRootRecord, error) {
	if !serviceBootIDPattern.MatchString(payloadBoot) || !serviceBootIDPattern.MatchString(currentBoot) {
		return windowsOwnedRootRecord{}, ErrOperationIdentity
	}
	if payloadBoot == currentBoot {
		return root, nil
	}
	if !jobAbsent {
		return windowsOwnedRootRecord{}, ErrOperationIdentity
	}
	return windowsOwnedRootRecord{}, nil
}
