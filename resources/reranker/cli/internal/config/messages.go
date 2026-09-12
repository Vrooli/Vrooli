package config

type Messages struct{ Healthy, Running, InstallFailed, HealthCheckFailed string }

func DefaultMessages() Messages {
	return Messages{Healthy: "Reranker API is healthy", Running: "Reranker service is running", InstallFailed: "Reranker installation failed", HealthCheckFailed: "Reranker health check failed"}
}
