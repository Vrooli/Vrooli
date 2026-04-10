package deployment

import (
	"testing"

	types "scenario-dependency-analyzer/internal/types"
)

func TestGetHealthEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *types.ServiceConfig
		serviceName string
		want        string
	}{
		{
			name:        "nil config returns default",
			cfg:         nil,
			serviceName: "api",
			want:        "/health",
		},
		{
			name: "nil lifecycle returns default",
			cfg: &types.ServiceConfig{
				Lifecycle: nil,
			},
			serviceName: "api",
			want:        "/health",
		},
		{
			name: "nil health config returns default",
			cfg: &types.ServiceConfig{
				Lifecycle: &types.ServiceLifecycle{
					Health: nil,
				},
			},
			serviceName: "api",
			want:        "/health",
		},
		{
			name: "empty endpoints map returns default",
			cfg: &types.ServiceConfig{
				Lifecycle: &types.ServiceLifecycle{
					Health: &types.ServiceHealthConfig{
						Endpoints: map[string]string{},
					},
				},
			},
			serviceName: "api",
			want:        "/health",
		},
		{
			name: "service not in endpoints returns default",
			cfg: &types.ServiceConfig{
				Lifecycle: &types.ServiceLifecycle{
					Health: &types.ServiceHealthConfig{
						Endpoints: map[string]string{
							"other": "/other-health",
						},
					},
				},
			},
			serviceName: "api",
			want:        "/health",
		},
		{
			name: "empty endpoint value returns default",
			cfg: &types.ServiceConfig{
				Lifecycle: &types.ServiceLifecycle{
					Health: &types.ServiceHealthConfig{
						Endpoints: map[string]string{
							"api": "",
						},
					},
				},
			},
			serviceName: "api",
			want:        "/health",
		},
		{
			name: "returns configured api endpoint",
			cfg: &types.ServiceConfig{
				Lifecycle: &types.ServiceLifecycle{
					Health: &types.ServiceHealthConfig{
						Endpoints: map[string]string{
							"api": "/api/healthz",
							"ui":  "/ui/health",
						},
					},
				},
			},
			serviceName: "api",
			want:        "/api/healthz",
		},
		{
			name: "returns configured ui endpoint",
			cfg: &types.ServiceConfig{
				Lifecycle: &types.ServiceLifecycle{
					Health: &types.ServiceHealthConfig{
						Endpoints: map[string]string{
							"api": "/health",
							"ui":  "/status",
						},
					},
				},
			},
			serviceName: "ui",
			want:        "/status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getHealthEndpoint(tt.cfg, tt.serviceName)
			if got != tt.want {
				t.Errorf("getHealthEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}
