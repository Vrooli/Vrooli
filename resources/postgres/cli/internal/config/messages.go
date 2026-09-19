package config

type Messages struct{ Healthy, Running, InstallFailed, HealthCheckFailed string }

func DefaultMessages() Messages {
	return Messages{Healthy: "PostgreSQL is healthy", Running: "PostgreSQL is running", InstallFailed: "PostgreSQL installation failed", HealthCheckFailed: "PostgreSQL health check failed"}
}
