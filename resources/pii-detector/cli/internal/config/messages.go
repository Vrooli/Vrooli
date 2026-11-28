package config

type Messages struct{ Healthy, Running, InstallFailed, HealthCheckFailed string }

func DefaultMessages() Messages {
	return Messages{Healthy: "PII detector API is healthy", Running: "PII detector service is running", InstallFailed: "PII detector installation failed", HealthCheckFailed: "PII detector health check failed"}
}
