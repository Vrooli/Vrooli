package platform

import "time"

// RootUpdateRecoveryDefinition is independent of the serving CLI, its source
// tree and pressure-report mode. It reconciles only an already-admitted root
// receipt, including after caller/process loss or a host reboot.
func RootUpdateRecoveryDefinition(target, home, binary string) (ServiceDefinition, error) {
	target, err := NormalizeTarget(target)
	if err != nil {
		return ServiceDefinition{}, err
	}
	if target != "linux" {
		return ServiceDefinition{}, ErrUnsupported
	}
	unit, _ := CoreUnitByID(CoreUnitRootUpdateRecovery)
	return ServiceDefinition{Name: "vrooli-root-update-recovery", Description: "Vrooli admitted root update recovery", DocumentationURL: DocumentationURL(unit.OwnerPath), Executable: binary, Args: []string{"--reconcile-core-updates"}, Env: map[string]string{"HOME": home, "PATH": DefaultPath(target, home)}, WorkingDirectory: home, Kind: KindTimer, Schedule: &Schedule{OnBoot: 15 * time.Second, Every: time.Minute, Persistent: true}, Restart: RestartPolicy{Mode: RestartNever}, Scope: ScopeUser, StartTimeout: 75 * time.Second, StopTimeout: 5 * time.Second, Protections: Protections{MemoryMin: "64M", OOMScoreAdjust: -500, Containment: Containment{MemoryMax: "512M", TasksMax: 128, CPUWeight: 400, ManagedOOMAvoid: true}}}, nil
}
