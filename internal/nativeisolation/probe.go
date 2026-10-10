package nativeisolation

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ProbeHelperPath is a reviewed nonmodel image executable. It must never be
// enrolled in Manifest.NativeBinaries as an arbitrary native launch target.
const ProbeHelperPath = "/usr/bin/vrooli-native-isolation-probe"

type ProbePurpose string

const (
	FirstQualification ProbePurpose = "first-qualification"
	ContinuingWitness  ProbePurpose = "continuing-witness"
)

// ProbeSpec is runtime-owner construction data, never HTTP input. BootstrapPlan
// constructs a fixed service definition; it starts nothing and does not complete
// H1 observation qualification. First qualification is limited to
// five minutes; a continuing witness cannot outlive the original policy or eight hours.
type ProbeSpec struct {
	Manifest         Manifest
	HelperSHA256     string
	Purpose          ProbePurpose
	Deadline         time.Time
	FirstObservation *FirstObservationContract
}

func (s ProbeSpec) validate(ctx context.Context, now time.Time) error {
	if ctx.Err() != nil || !s.Manifest.Enabled || s.Manifest.Validate() != nil || !validHash(s.HelperSHA256) || !now.Before(s.Deadline) {
		return ErrRefused
	}
	if _, model := s.Manifest.NativeBinaries[ProbeHelperPath]; model {
		return ErrRefused
	}
	if s.Purpose == FirstQualification && (len(s.Manifest.NetworkAllow) != 0 || s.FirstObservation == nil) {
		return ErrRefused
	}
	limit := 5 * time.Minute
	if s.Purpose == ContinuingWitness {
		limit = 8 * time.Hour
	} else if s.Purpose != FirstQualification {
		return ErrRefused
	}
	if s.Purpose == ContinuingWitness && s.FirstObservation != nil {
		return ErrRefused
	}
	if s.Purpose == FirstQualification {
		if _, e := s.observation(now); e != nil {
			return ErrRefused
		}
	}
	if s.Deadline.Sub(now) > limit {
		return ErrRefused
	}
	for _, b := range s.Manifest.Bindings {
		if s.Deadline.After(b.Deadline) {
			return ErrRefused
		}
	}
	return nil
}

func (s ProbeSpec) unit() string { return "vrooli-finite-probe-" + digest(s) + ".service" }

// BootstrapPlan has no arbitrary command, arguments, input, network or model.
// Execution must rebuild this plan from the installed spec before any effect;
// exported Plan fields are review data and are not execution authority.
func (s ProbeSpec) BootstrapPlan(ctx context.Context, now time.Time) (Plan, error) {
	if s.validate(ctx, now) != nil {
		return Plan{}, ErrRefused
	}
	m := s.Manifest
	repository := s.repository()
	seconds := int64(s.Deadline.Sub(now) / time.Second)
	if seconds < 1 {
		return Plan{}, ErrRefused
	}
	properties := []string{
		"User=" + fmt.Sprint(m.NativeUID), "Group=" + fmt.Sprint(m.NativeGID),
		"RootDirectory=" + m.RootFS, "WorkingDirectory=" + repository,
		"BindPaths=" + m.WorkspaceSource + ":" + repository, "ReadWritePaths=" + repository, "ProtectSystem=strict", "ProtectHome=tmpfs",
		"PrivateMounts=yes", "PrivateDevices=yes", "MountAPIVFS=yes", "ProtectProc=invisible", "ProcSubset=pid",
		"NoNewPrivileges=yes", "CapabilityBoundingSet=", "AmbientCapabilities=", "RestrictSUIDSGID=yes",
		"RestrictNamespaces=yes", "Delegate=no", "KillMode=control-group", "Restart=no",
		"ProtectKernelTunables=yes", "ProtectKernelModules=yes", "ProtectKernelLogs=yes", "ProtectControlGroups=yes",
		"RestrictAddressFamilies=AF_INET AF_INET6", "SystemCallFilter=~@mount @privileged @reboot @swap",
		"TemporaryFileSystem=/tmp:rw,nosuid,nodev /run:rw,nosuid,nodev", "IPAddressDeny=any",
		"RuntimeMaxSec=" + fmt.Sprint(seconds), "TimeoutStopSec=5s",
	}
	if len(m.NetworkAllow) > 0 {
		properties = append(properties, "IPAddressAllow="+strings.Join(m.NetworkAllow, " "))
	}
	if s.Purpose == FirstQualification {
		properties = append(properties, "BindReadOnlyPaths="+s.FirstObservation.deniedSource()+":/vrooli-qualification/denied.marker")
	}
	args := []string{"--system", "--quiet", "--service-type=exec", "--unit=" + s.unit()}
	for _, p := range properties {
		args = append(args, "--property="+p)
	}
	if s.Purpose == FirstQualification {
		o, _ := s.observation(now)
		encoded, e := o.Encode(now)
		if e != nil {
			return Plan{}, ErrRefused
		}
		args = append(args, "--", ProbeHelperPath, "--first", "--contract", encoded)
	} else {
		args = append(args, "--", ProbeHelperPath, "--until", s.Deadline.UTC().Format(time.RFC3339Nano))
	}
	p := Plan{Unit: s.unit(), Command: m.SystemdRun.Path, Args: args, Env: []string{"LANG=C", "LC_ALL=C"}, Digest: digest(s)}
	return p, nil
}

// repository is fixed by the sole installed policy binding validated above.
func (s ProbeSpec) repository() string {
	for _, binding := range s.Manifest.Bindings {
		return binding.Repository
	}
	return ""
}
