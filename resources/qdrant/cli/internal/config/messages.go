package config

type Messages struct{ Healthy, Running, InstallFailed, HealthCheckFailed string }

func DefaultMessages() Messages {
	return Messages{Healthy: "Qdrant API is healthy", Running: "Qdrant service is running", InstallFailed: "Qdrant installation failed", HealthCheckFailed: "Qdrant health check failed"}
}
