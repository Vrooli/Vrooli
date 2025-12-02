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
	m := &manifest.Manifest{}
	pm := NewPortManager(m, RealNetworkDialer{})
	// Manually set up port map for testing
	pm.mu.Lock()
	pm.portMap = map[string]map[string]int{
		"api":    {"http": 47000, "grpc": 47001},
		"worker": {"metrics": 47002},
	}
	pm.mu.Unlock()

	tests := []struct {
		name      string
		serviceID string
		portName  string
		want      int
		wantErr   bool
	}{
		{"existing port", "api", "http", 47000, false},
		{"second port", "api", "grpc", 47001, false},
		{"different service", "worker", "metrics", 47002, false},
		{"unknown service", "unknown", "http", 0, true},
		{"unknown port", "api", "unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pm.Resolve(tt.serviceID, tt.portName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortManagerPickPort(t *testing.T) {
	tests := []struct {
		name     string
		rng      PortRange
		reserved map[int]bool
		next     int
		wantErr  bool
	}{
		{
			name:     "valid range",
			rng:      PortRange{Min: 47000, Max: 47010},
			reserved: map[int]bool{},
			next:     47000,
			wantErr:  false,
		},
		{
			name:     "skips reserved",
			rng:      PortRange{Min: 47000, Max: 47010},
			reserved: map[int]bool{47000: true, 47001: true},
			next:     47000,
			wantErr:  false,
		},
		{
			name:     "invalid range min > max",
			rng:      PortRange{Min: 47010, Max: 47000},
			reserved: map[int]bool{},
			next:     47000,
			wantErr:  true,
		},
		{
			name:     "zero range",
			rng:      PortRange{Min: 0, Max: 0},
			reserved: map[int]bool{},
			next:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPortManager(&manifest.Manifest{}, RealNetworkDialer{})
			next := tt.next
			port, err := pm.pickPort(tt.rng, tt.reserved, &next)
			if (err != nil) != tt.wantErr {
				t.Errorf("pickPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if port < tt.rng.Min || port > tt.rng.Max {
					t.Errorf("pickPort() = %v, outside range %d-%d", port, tt.rng.Min, tt.rng.Max)
				}
				if tt.reserved[port] {
					t.Errorf("pickPort() = %v, but port is reserved", port)
				}
			}
		})
	}
}

func TestPortManagerMap(t *testing.T) {
	m := &manifest.Manifest{}
	pm := NewPortManager(m, RealNetworkDialer{})
	// Manually set up port map for testing
	pm.mu.Lock()
	pm.portMap = map[string]map[string]int{
		"api": {"http": 47000},
	}
	pm.mu.Unlock()

	result := pm.Map()

	// Verify it's a copy
	result["api"]["http"] = 99999
	original := pm.Map()
	if original["api"]["http"] == 99999 {
		t.Error("Map() returned reference instead of copy")
	}

	// Verify original value
	if original["api"]["http"] != 47000 {
		t.Errorf("original portMap modified, got %d want 47000", original["api"]["http"])
	}
}

// Note: testMockPortAllocator is defined in health_test.go and reused here.
