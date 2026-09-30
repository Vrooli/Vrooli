package resources

import (
	"context"
	"testing"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestNativeManagedSharedBootstrapRequiresSharedInstance(t *testing.T) {
	instance := ManagedInstance{Resource: "minio", Provider: resourcedeployment.ProviderManagedPrivate}
	if err := nativeManagedSharedBootstrap(context.Background(), nil, instance, "control-plane"); err == nil {
		t.Fatal("native bootstrap accepted a private instance")
	}
	instance.Provider = resourcedeployment.ProviderManagedShared
	instance.Resource = ""
	if err := nativeManagedSharedBootstrap(context.Background(), nil, instance, "control-plane"); err == nil {
		t.Fatal("native bootstrap accepted an instance without a resource")
	}
	instance.Resource = "minio"
	if err := nativeManagedSharedBootstrap(context.Background(), nil, instance, "control-plane"); err != nil {
		t.Fatalf("native bootstrap rejected a valid shared instance: %v", err)
	}
}

func TestManagedServiceLoopbackEndpointPrefersHTTPThenAPI(t *testing.T) {
	manifest := ResourceManifest{Name: "minio", Ports: []ResourcePort{{Name: "api", Host: 9000}}}
	endpoint, err := managedServiceLoopbackEndpoint(manifest)
	if err != nil || endpoint != "http://127.0.0.1:9000" {
		t.Fatalf("endpoint=%q err=%v", endpoint, err)
	}
	manifest.Ports = []ResourcePort{{Name: "grpc", Host: 6334}, {Name: "http", Host: 6333}}
	endpoint, err = managedServiceLoopbackEndpoint(manifest)
	if err != nil || endpoint != "http://127.0.0.1:6333" {
		t.Fatalf("preferred endpoint=%q err=%v", endpoint, err)
	}
}
