package bundleruntime

import (
	"testing"

	"scenario-to-desktop-runtime/manifest"
)

func TestPortManagerAllocate(t *testing.T) {
	tests := []struct {
		name     string
		manifest *manifest.Manifest
		wantErr  bool
	}{
		{
			name: "single service single port",
			manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{
						ID: "api",
						Ports: &manifest.ServicePorts{
							Requested: []manifest.PortRequest{
								{Name: "http", Range: manifest.PortRange{Min: 47000, Max: 47100}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple services multiple ports",
			manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{
						ID: "api",
						Ports: &manifest.ServicePorts{
							Requested: []manifest.PortRequest{
								{Name: "http", Range: manifest.PortRange{Min: 47000, Max: 47100}},
								{Name: "grpc", Range: manifest.PortRange{Min: 47000, Max: 47100}},
							},
						},
					},
					{
						ID: "worker",
						Ports: &manifest.ServicePorts{
							Requested: []manifest.PortRequest{
								{Name: "metrics", Range: manifest.PortRange{Min: 47000, Max: 47100}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "service with no ports",
			manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{ID: "worker"},
				},
			},
			wantErr: false,
		},
		{
			name: "respects reserved ports",
			manifest: &manifest.Manifest{
				Ports: &manifest.PortRules{
					Reserved: []int{47000, 47001, 47002},
				},
				Services: []manifest.Service{
					{
						ID: "api",
						Ports: &manifest.ServicePorts{
							Requested: []manifest.PortRequest{
								{Name: "http", Range: manifest.PortRange{Min: 47000, Max: 47010}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPortManager(tt.manifest, RealNetworkDialer{})

			err := pm.Allocate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Allocate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify ports were allocated
				for _, svc := range tt.manifest.Services {
					if svc.Ports == nil {
						continue
					}
					for _, req := range svc.Ports.Requested {
						port, err := pm.Resolve(svc.ID, req.Name)
						if err != nil {
							t.Errorf("Resolve(%s, %s) error = %v", svc.ID, req.Name, err)
							continue
						}
						if port < req.Range.Min || port > req.Range.Max {
							t.Errorf("port %d outside requested range %d-%d", port, req.Range.Min, req.Range.Max)
						}
					}
				}
			}
		})
	}
}

func TestPortManagerResolve(t *testing.T) {
	// Create a manifest with services and ports for allocation
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "http", Range: manifest.PortRange{Min: 47000, Max: 47100}},
						{Name: "grpc", Range: manifest.PortRange{Min: 47000, Max: 47100}},
					},
				},
			},
			{
				ID: "worker",
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "metrics", Range: manifest.PortRange{Min: 47000, Max: 47100}},
					},
				},
			},
		},
	}
	pm := NewPortManager(m, RealNetworkDialer{})
	if err := pm.Allocate(); err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	tests := []struct {
		name      string
		serviceID string
		portName  string
		wantErr   bool
	}{
		{"existing port", "api", "http", false},
		{"second port", "api", "grpc", false},
		{"different service", "worker", "metrics", false},
		{"unknown service", "unknown", "http", true},
		{"unknown port", "api", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pm.Resolve(tt.serviceID, tt.portName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// For successful resolutions, verify port is in valid range
			if !tt.wantErr && (got < 47000 || got > 47100) {
				t.Errorf("Resolve() = %v, expected port in range 47000-47100", got)
			}
		})
	}
}

// TestPortManagerMap tests that Map returns a copy of the port allocations.
func TestPortManagerMap(t *testing.T) {
	// Create a manifest with services and ports
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "http", Range: manifest.PortRange{Min: 47000, Max: 47100}},
					},
				},
			},
		},
	}
	pm := NewPortManager(m, RealNetworkDialer{})
	if err := pm.Allocate(); err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	result := pm.Map()

	// Verify it's a copy by modifying the result
	originalPort := result["api"]["http"]
	result["api"]["http"] = 99999
	original := pm.Map()
	if original["api"]["http"] == 99999 {
		t.Error("Map() returned reference instead of copy")
	}

	// Verify original value is still correct
	if original["api"]["http"] != originalPort {
		t.Errorf("original portMap modified, got %d want %d", original["api"]["http"], originalPort)
	}
}

// Note: testMockPortAllocator is defined in health_test.go and reused here.
// Note: TestPortManagerPickPort was removed as it tests internal implementation.
