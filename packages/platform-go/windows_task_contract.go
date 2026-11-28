package platform

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// WindowsTaskSnapshot describes manager evidence. EnginePID belongs to the
// scheduler engine; it is intentionally never exposed as an action/root PID.
type WindowsTaskSnapshot struct {
	Name, XML, SecurityDescriptor string
	State                         int32
	Enabled                       bool
	Instances                     []WindowsTaskInstance
}
type WindowsTaskInstance struct {
	GUID      string
	EnginePID uint32
	State     int32
}

type windowsTaskDefinition struct {
	XMLName    xml.Name
	Principals []struct {
		UserID    string `xml:"UserId"`
		LogonType string `xml:"LogonType"`
		RunLevel  string `xml:"RunLevel"`
	} `xml:"Principals>Principal"`
	Boot     []windowsTaskBootTrigger `xml:"Triggers>BootTrigger"`
	Time     []windowsTaskTimeTrigger `xml:"Triggers>TimeTrigger"`
	Settings struct {
		Limit          string `xml:"ExecutionTimeLimit"`
		HardTerminate  string `xml:"AllowHardTerminate"`
		Multiple       string `xml:"MultipleInstancesPolicy"`
		NoBatteryStart string `xml:"DisallowStartIfOnBatteries"`
		StopOnBattery  string `xml:"StopIfGoingOnBatteries"`
		Demand         string `xml:"AllowStartOnDemand"`
		Network        string `xml:"RunOnlyIfNetworkAvailable"`
		Idle           string `xml:"RunOnlyIfIdle"`
		Volatile       string `xml:"Volatile"`
		RemoteApp      string `xml:"DisallowStartOnRemoteAppSession"`
		StartAvailable string `xml:"StartWhenAvailable"`
		Restart        struct {
			Interval string `xml:"Interval"`
			Count    string `xml:"Count"`
		} `xml:"RestartOnFailure"`
	} `xml:"Settings"`
	Actions []struct {
		Command   string `xml:"Command"`
		Arguments string `xml:"Arguments"`
		Directory string `xml:"WorkingDirectory"`
	} `xml:"Actions>Exec"`
}

type windowsTaskBootTrigger struct {
	Enabled string `xml:"Enabled"`
	Delay   string `xml:"Delay"`
}
type windowsTaskTimeTrigger struct {
	Enabled    string `xml:"Enabled"`
	Start      string `xml:"StartBoundary"`
	Repetition struct {
		Interval string `xml:"Interval"`
		Stop     string `xml:"StopAtDurationEnd"`
	} `xml:"Repetition"`
}

// A protected definition must still prove headless daemon policy; accepting
// its image and principal alone does not establish boot or continuous running.
func windowsOperatorBootPolicy(d windowsTaskDefinition) bool {
	return len(d.Boot) == 1 && d.Boot[0].Enabled == "true" && d.Boot[0].Delay == "PT30S" && len(d.Time) == 0 &&
		d.Settings.Limit == "PT0S" && d.Settings.Multiple == "IgnoreNew" && d.Settings.NoBatteryStart == "false" &&
		d.Settings.StopOnBattery == "false" && d.Settings.Demand == "true" && d.Settings.StartAvailable == "true" && d.Settings.Restart.Interval == "PT1M" && d.Settings.Restart.Count == "999" && d.Settings.RemoteApp == "false" &&
		d.Settings.Network == "false" &&
		d.Settings.Idle == "false" &&
		d.Settings.Volatile == "false"
}

func parseWindowsTaskDefinition(content string) (windowsTaskDefinition, error) {
	var d windowsTaskDefinition
	if len(content) > 1<<20 {
		return d, errors.New("platform: task XML exceeds bound")
	}
	if e := validateWindowsTaskElements(content); e != nil {
		return d, e
	}
	decoder := xml.NewDecoder(strings.NewReader(content))
	if e := decoder.Decode(&d); e != nil {
		return d, e
	}
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return d, e
		}
		if chars, ok := token.(xml.CharData); !ok || strings.TrimSpace(string(chars)) != "" {
			return d, errors.New("platform: trailing task XML")
		}
	}
	if d.XMLName.Local != "Task" || d.XMLName.Space != windowsTaskNamespace || len(d.Principals) != 1 || len(d.Actions) != 1 || d.Principals[0].UserID == "" || d.Principals[0].LogonType != "S4U" || d.Principals[0].RunLevel != "LeastPrivilege" {
		return d, fmt.Errorf("platform: task requires one S4U least-privilege principal and one Exec action: %w", ErrOperationIdentity)
	}
	if d.Settings.HardTerminate == "" {
		d.Settings.HardTerminate = "true"
	}
	// The scheduler may materialize documented false defaults in exports.
	// Canonicalize only these defaults so observation does not reject its own task.
	if d.Settings.RemoteApp == "" {
		d.Settings.RemoteApp = "false"
	}
	if d.Settings.Network == "" {
		d.Settings.Network = "false"
	}
	if d.Settings.Idle == "" {
		d.Settings.Idle = "false"
	}
	if d.Settings.Volatile == "" {
		d.Settings.Volatile = "false"
	}
	return d, nil
}

// Reject action kinds and schema extensions we do not own before registration.
func validateWindowsTaskElements(content string) error {
	allowed := map[string]string{
		"": "Task", "Task": "RegistrationInfo Triggers Principals Settings Actions", "Task/RegistrationInfo": "Description Author Documentation URI Date SecurityDescriptor Source Version", "Task/Triggers": "BootTrigger TimeTrigger", "Task/Triggers/BootTrigger": "Enabled Delay", "Task/Triggers/TimeTrigger": "Enabled StartBoundary Repetition", "Task/Triggers/TimeTrigger/Repetition": "Interval StopAtDurationEnd", "Task/Principals": "Principal", "Task/Principals/Principal": "UserId LogonType RunLevel", "Task/Settings": "MultipleInstancesPolicy DisallowStartIfOnBatteries StopIfGoingOnBatteries AllowHardTerminate StartWhenAvailable AllowStartOnDemand Enabled Hidden Priority ExecutionTimeLimit RestartOnFailure RunOnlyIfNetworkAvailable RunOnlyIfIdle WakeToRun IdleSettings UseUnifiedSchedulingEngine DisallowStartOnRemoteAppSession Volatile Compatibility", "Task/Settings/RestartOnFailure": "Interval Count", "Task/Settings/IdleSettings": "Duration WaitTimeout StopOnIdleEnd RestartOnIdle", "Task/Actions": "Exec", "Task/Actions/Exec": "Command Arguments WorkingDirectory",
	}
	decoder := xml.NewDecoder(strings.NewReader(content))
	var stack []string
	seen := map[string]bool{}
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return e
		}
		switch v := token.(type) {
		case xml.StartElement:
			parent := strings.Join(stack, "/")
			if v.Name.Space != windowsTaskNamespace || !strings.Contains(" "+allowed[parent]+" ", " "+v.Name.Local+" ") {
				return fmt.Errorf("platform: unowned task XML element %s/%s", parent, v.Name.Local)
			}
			path := parent + "/" + v.Name.Local
			if seen[path] {
				return errors.New("platform: duplicate task XML element")
			}
			seen[path] = true
			for _, a := range v.Attr {
				valid := (len(stack) == 0 && ((a.Name.Space == "" && a.Name.Local == "xmlns") || (a.Name.Space == "" && a.Name.Local == "version"))) || (parent == "Task/Principals" && a.Name.Space == "" && a.Name.Local == "id") || (parent == "Task" && v.Name.Local == "Actions" && a.Name.Space == "" && a.Name.Local == "Context")
				if !valid {
					return errors.New("platform: unowned task XML attribute")
				}
			}
			stack = append(stack, v.Name.Local)
		case xml.EndElement:
			if len(stack) == 0 {
				return ErrOperationIdentity
			}
			stack = stack[:len(stack)-1]
		case xml.Directive:
			return errors.New("platform: task XML directives forbidden")
		case xml.ProcInst:
			if v.Target != "xml" || len(stack) != 0 {
				return ErrOperationIdentity
			}
		}
	}
}

func windowsTaskSameIncarnation(before, after WindowsTaskSnapshot) bool {
	return before.Name == after.Name && before.XML == after.XML && before.SecurityDescriptor == after.SecurityDescriptor && before.State == 4 && after.State == 4 && len(before.Instances) == 1 && len(after.Instances) == 1 && before.Instances[0] == after.Instances[0]
}
func windowsTaskActionRelation(actionPID, parentPID uint32, actionBirth uint64, instance WindowsTaskInstance, engineBirth uint64) bool {
	if actionPID == 0 || instance.EnginePID == 0 || instance.GUID == "" || instance.State != 4 || actionBirth == 0 || engineBirth == 0 {
		return false
	}
	if actionPID == instance.EnginePID {
		return actionBirth == engineBirth
	}
	return parentPID == instance.EnginePID && actionBirth >= engineBirth
}

// Only the documented Win32 missing-file HRESULT proves task absence.
// SCHED_E_ACCOUNT_INFORMATION_NOT_SET (0x8004130f) is an existing task
// security/configuration failure, never absence.
func windowsTaskHRESULTMissing(hr uint32) bool { return hr == 0x80070002 }

// Native IRegisteredTask.Stop acknowledges cancellation only with S_OK.
// Its documented S_FALSE means instances could not be stopped.
func windowsTaskStopAcknowledged(hr uint32) bool { return hr == 0 }

// This error describes no running instance, not cancellation of queued work.
// It permits idempotent Stop only with a fresh positive quiescence observation.
func windowsTaskHRESULTNotRunning(hr uint32) bool { return hr == 0x8004130b }
