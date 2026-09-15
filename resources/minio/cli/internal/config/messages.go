package config

type Messages struct{ Healthy, Running, InstallFailed, HealthCheckFailed string }

func DefaultMessages() Messages {
	return Messages{Healthy: "MinIO API is healthy", Running: "MinIO service is running", InstallFailed: "MinIO installation failed", HealthCheckFailed: "MinIO health check failed"}
}
