package destinationtopology

import "testing"

func TestCompareFailureDomains(t *testing.T) {
	tests := []struct {
		name string
		a, b Evidence
		want Relation
	}{
		{"same disk", Evidence{HostID: "host-a", DeviceID: "disk-1", Endpoint: EndpointLocal}, Evidence{HostID: "host-a", DeviceID: "disk-1", Endpoint: EndpointLocal}, RelationSameDevice},
		{"same host different disks", Evidence{HostID: "host-a", DeviceID: "disk-1", Endpoint: EndpointLocal}, Evidence{HostID: "host-a", DeviceID: "disk-2", Endpoint: EndpointLocal}, RelationSameHost},
		{"different hosts", Evidence{HostID: "host-a", DeviceID: "disk-1", Endpoint: EndpointLocal}, Evidence{HostID: "host-b", DeviceID: "disk-2", Endpoint: EndpointRemoteHost}, RelationIndependent},
		{"S3 local endpoint", Evidence{HostID: "host-a", Endpoint: EndpointLocal}, Evidence{HostID: "host-a", Endpoint: EndpointLocal}, RelationSameHost},
		{"provider hosted S3 vs local", Evidence{HostID: "host-a", Endpoint: EndpointLocal}, Evidence{Endpoint: EndpointProviderHosted}, RelationIndependent},
		{"S3 kind without topology", Evidence{}, Evidence{Endpoint: EndpointUnknown}, RelationUnknown},
		{"unknown topology", Evidence{}, Evidence{}, RelationUnknown},
		{"remote host without identity", Evidence{HostID: "host-a", Endpoint: EndpointLocal}, Evidence{Endpoint: EndpointRemoteHost}, RelationUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compare(tt.a, tt.b); got != tt.want {
				t.Fatalf("Compare() = %q, want %q", got, tt.want)
			}
		})
	}
}
