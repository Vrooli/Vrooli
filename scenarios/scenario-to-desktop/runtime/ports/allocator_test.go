package ports_test

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/ports"
)

func TestManagerAllocate(t *testing.T) {
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
			pm := ports.NewManager(tt.manifest, infra.RealNetworkDialer{})

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

func TestManagerResolve(t *testing.T) {
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
	pm := ports.NewManager(m, infra.RealNetworkDialer{})
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

// TestManagerMap tests that Map returns a copy of the port allocations.
func TestManagerMap(t *testing.T) {
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
	pm := ports.NewManager(m, infra.RealNetworkDialer{})
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

// TestManagerSetPorts tests that SetPorts allows directly setting the port map.
func TestManagerSetPorts(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "api"},
			{ID: "worker"},
		},
	}
	pm := ports.NewManager(m, infra.RealNetworkDialer{})

	// Set ports directly without calling Allocate
	customPorts := map[string]map[string]int{
		"api":    {"http": 8080, "grpc": 9090},
		"worker": {"metrics": 9100},
	}
	pm.SetPorts(customPorts)

	// Verify the ports were set correctly
	port, err := pm.Resolve("api", "http")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if port != 8080 {
		t.Errorf("Resolve(api, http) = %d, want 8080", port)
	}

	port, err = pm.Resolve("api", "grpc")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if port != 9090 {
		t.Errorf("Resolve(api, grpc) = %d, want 9090", port)
	}

	port, err = pm.Resolve("worker", "metrics")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if port != 9100 {
		t.Errorf("Resolve(worker, metrics) = %d, want 9100", port)
	}

	// Verify Map returns the same data
	result := pm.Map()
	if len(result) != 2 {
		t.Errorf("Map() returned %d services, want 2", len(result))
	}
	if result["api"]["http"] != 8080 {
		t.Errorf("Map()[api][http] = %d, want 8080", result["api"]["http"])
	}
}

func TestManagerAllocateFailsWhenRangeExhausted(t *testing.T) {
	manifestWithTinyRange := &manifest.Manifest{
		Ports: &manifest.PortRules{
			DefaultRange: &manifest.PortRange{Min: 47000, Max: 47001},
		},
		Services: []manifest.Service{
			{
				ID: "api",
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "http"},
					},
				},
			},
		},
	}

	dialer := &stubDialer{
		availability: map[int]bool{
			47000: false,
			47001: false,
		},
	}

	pm := ports.NewManager(manifestWithTinyRange, dialer)
	err := pm.Allocate()
	if err == nil {
		t.Fatalf("Allocate() expected error when range is exhausted")
	}
	if !strings.Contains(err.Error(), "no free port in 47000-47001") {
		t.Fatalf("Allocate() error = %v, want message about exhausted range", err)
	}
}

func TestManagerAllocateRejectsInvalidRequestedRange(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "http", Range: manifest.PortRange{Min: 48000, Max: 47000}},
					},
				},
			},
		},
	}
	pm := ports.NewManager(m, &stubDialer{availability: map[int]bool{}})

	err := pm.Allocate()
	if err == nil {
		t.Fatalf("Allocate() expected error for invalid requested range")
	}
	if !strings.Contains(err.Error(), "invalid range 48000-47000") {
		t.Fatalf("Allocate() error = %v, want invalid range message", err)
	}
}

type stubDialer struct {
	availability map[int]bool
}

func (s *stubDialer) Dial(network, address string) (net.Conn, error) {
	return nil, errors.New("Dial not implemented")
}

func (s *stubDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return nil, errors.New("DialTimeout not implemented")
}

func (s *stubDialer) Listen(network, address string) (net.Listener, error) {
	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("split host port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("parse port: %w", err)
	}
	if s.availability[port] {
		return &noopListener{}, nil
	}
	return nil, fmt.Errorf("port %d unavailable", port)
}

type noopListener struct{}

func (n *noopListener) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (n *noopListener) Close() error              { return nil }
func (n *noopListener) Addr() net.Addr            { return &net.TCPAddr{} }
