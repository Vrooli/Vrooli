package support

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
)

const (
	testPermissionRead  = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission"
	testPermissionWrite = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission"
)

func TestWriteCurrentPermissionBindsOwnerRevisionAndReceipt(t *testing.T) {
	const permissionID = "onboarding/host-safeguard/firewall"
	value := []byte(`{"config":{"enabled":true}}`)
	var writeRequest struct {
		PermissionID     string `json:"permissionId"`
		Owner            string `json:"owner"`
		ExpectedRevision string `json:"expectedRevision"`
		IdempotencyKey   string `json:"idempotencyKey"`
		ValueJSON        string `json:"valueJson"`
	}
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPermissionRead:
			var request struct {
				PermissionID string `json:"permissionId"`
				Owner        string `json:"owner"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.PermissionID != permissionID || request.Owner != "vrooli-onboarding" {
				t.Fatalf("read request = %#v", request)
			}
			_, _ = w.Write([]byte(`{"permissionId":"onboarding/host-safeguard/firewall","owner":"vrooli-onboarding","revision":"source-r7"}`))
		case testPermissionWrite:
			if err := json.NewDecoder(r.Body).Decode(&writeRequest); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"receipt":{"permissionId":"onboarding/host-safeguard/firewall","owner":"vrooli-onboarding","actorId":"operator-1","expectedRevision":"source-r7","resultRevision":"source-r8","idempotencyKey":"` + writeRequest.IdempotencyKey + `","outcome":"committed"}}`))
		default:
			t.Fatalf("unexpected permission path %q", r.URL.Path)
		}
	}))

	receipt, err := WriteCurrentPermission(core, permissionID, "vrooli-onboarding", "cli-host-safeguard", "host safeguard permission", nil, value)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.GetActorId() != "operator-1" || receipt.GetExpectedRevision() != "source-r7" || receipt.GetResultRevision() != "source-r8" {
		t.Fatalf("receipt = %#v", receipt)
	}
	decodedValue, decodeErr := base64.StdEncoding.DecodeString(writeRequest.ValueJSON)
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if writeRequest.PermissionID != permissionID || writeRequest.Owner != "vrooli-onboarding" || writeRequest.ExpectedRevision != "source-r7" ||
		!strings.HasPrefix(writeRequest.IdempotencyKey, "cli-host-safeguard:") || !reflect.DeepEqual(decodedValue, value) {
		t.Fatalf("write request = %#v", writeRequest)
	}
}

func TestWriteCurrentPermissionRefusesMissingRevisionBeforeWrite(t *testing.T) {
	writes := 0
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPermissionWrite {
			writes++
		}
		_, _ = w.Write([]byte(`{"permissionId":"onboarding/host-safeguard/firewall","owner":"vrooli-onboarding"}`))
	}))
	if _, err := WriteCurrentPermission(core, "onboarding/host-safeguard/firewall", "vrooli-onboarding", "cli-host-safeguard", "host safeguard permission", nil, []byte(`{"opted_in":true}`)); err == nil {
		t.Fatal("missing revision was accepted")
	}
	if writes != 0 {
		t.Fatalf("writes = %d, want zero", writes)
	}
}

func TestWriteCurrentPermissionRefusesMismatchedReceipt(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPermissionRead {
			_, _ = w.Write([]byte(`{"permissionId":"onboarding/host-safeguard/firewall","owner":"vrooli-onboarding","revision":"r1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"receipt":{"permissionId":"other","owner":"vrooli-onboarding","actorId":"operator-1","expectedRevision":"r1","resultRevision":"r2","idempotencyKey":"wrong","outcome":"committed"}}`))
	}))
	if _, err := WriteCurrentPermission(core, "onboarding/host-safeguard/firewall", "vrooli-onboarding", "cli-host-safeguard", "host safeguard permission", nil, []byte(`{"opted_in":true}`)); err == nil {
		t.Fatal("mismatched receipt was accepted")
	}
}

func TestWriteCurrentPermissionPreservesTargetGeneration(t *testing.T) {
	targetJSON := `{"machineId":"machine-a","nodeId":"node-a","enrollmentGeneration":"9"}`
	target := &permissionsv1.PermissionTarget{MachineId: "machine-a", NodeId: "node-a", EnrollmentGeneration: 9}
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPermissionRead {
			_, _ = w.Write([]byte(`{"permissionId":"owner/target-permission","owner":"sample-owner","target":` + targetJSON + `,"revision":"target-r3"}`))
			return
		}
		var request struct {
			Target         json.RawMessage `json:"target"`
			IdempotencyKey string          `json:"idempotencyKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		var receivedTarget struct {
			MachineID            string `json:"machineId"`
			NodeID               string `json:"nodeId"`
			EnrollmentGeneration string `json:"enrollmentGeneration"`
		}
		if err := json.Unmarshal(request.Target, &receivedTarget); err != nil {
			t.Fatal(err)
		}
		if receivedTarget.MachineID != target.GetMachineId() || receivedTarget.NodeID != target.GetNodeId() || receivedTarget.EnrollmentGeneration != "9" {
			t.Fatalf("write target = %#v", receivedTarget)
		}
		body, err := json.Marshal(map[string]any{"receipt": map[string]any{
			"permissionId": "owner/target-permission", "owner": "sample-owner", "actorId": "operator-1",
			"target": request.Target, "expectedRevision": "target-r3", "resultRevision": "target-r4",
			"idempotencyKey": request.IdempotencyKey, "outcome": "committed",
		}})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(body)
	}))

	receipt, err := WriteCurrentPermission(core, "owner/target-permission", "sample-owner", "sample", "target permission", target, []byte(`{"enabled":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !samePermissionTarget(receipt.GetTarget(), target) || receipt.GetExpectedRevision() != "target-r3" {
		t.Fatalf("target receipt = %#v", receipt)
	}
}
