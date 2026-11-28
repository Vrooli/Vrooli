package bundles

// validBundleTestManifest returns fresh nested data for tests that exercise a
// complete, ordinary desktop bundle. Cases that test preservation or policy
// override only the fields that make that contract distinct.
func validBundleTestManifest() Manifest {
	return Manifest{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App:           ManifestApp{Name: "demo", Version: "1.0.0"},
		IPC:           ManifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 48000, AuthTokenPath: "runtime/auth-token"},
		Telemetry:     ManifestTelemetry{File: "telemetry.jsonl"},
		Ports:         &ManifestPorts{DefaultRange: PortRange{Min: 47000, Max: 47999}, Reserved: []int{}},
		Services: []ServiceEntry{{
			ID:        "api",
			Type:      "api-binary",
			Binaries:  map[string]ServiceBinary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp", PortName: "http", Interval: 1000, Timeout: 1000, Retries: 1},
			Readiness: ReadinessCheck{Type: "port_open", PortName: "http", Timeout: 1000},
			Ports:     &ServicePorts{Requested: []RequestedPort{{Name: "http", Range: PortRange{Min: 48000, Max: 48010}}}},
		}},
	}
}

func analyzerTestSkeleton(appName, version, description, serviceDescription string) Manifest {
	manifest := validBundleTestManifest()
	manifest.App.Name = appName
	manifest.App.Version = version
	manifest.App.Description = description
	manifest.Services[0].Description = serviceDescription
	manifest.Services[0].LogDir = "logs/api"
	manifest.Services[0].Health = HealthCheck{Type: "tcp", PortName: "http", Interval: 1000, Timeout: 2000, Retries: 3}
	manifest.Services[0].Readiness.Timeout = 5000
	manifest.IPC.Port = 48000
	manifest.Telemetry.File = "telemetry/deployment-telemetry.jsonl"
	return manifest
}
