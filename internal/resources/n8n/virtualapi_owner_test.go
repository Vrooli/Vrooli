package n8n

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStartVirtualAPIContractRejectsDuplicateFieldsBeforeOwnerCalls(t *testing.T) {
	runner := &fakeRunner{}
	owner := &Owner{Runner: runner}
	contract := []byte(`{"taskId":"` + testSandboxID + `","taskId":"` + testSandboxID + `","hosts":["api.notion.com"],"routes":[{"host":"api.notion.com","method":"GET","path":"/v1/users","status":200,"contentType":"application/json","body":"e30="}]}`)
	if _, err := owner.StartVirtualAPIContract(context.Background(), testSandboxID, contract); err == nil {
		t.Fatal("accepted a duplicate task ID")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("invalid contract reached the Docker owner: %#v", runner.calls)
	}
}

func TestVirtualAPIStagesTrustBeforeServerProgramStarts(t *testing.T) {
	files := virtualAPIStagedFiles([]byte("config"), []byte("task CA"), []byte("server"))
	want := []string{"/tmp/virtualapi-config.json", "/tmp/virtualapi-ca.pem", "/tmp/virtualapi-server.js", "/tmp/virtualapi-start"}
	if len(files) != len(want) {
		t.Fatalf("staged file count=%d want %d", len(files), len(want))
	}
	for index, path := range want {
		if files[index].path != path {
			t.Fatalf("stage order=%v want %v", files, want)
		}
	}
	for _, file := range files[:len(files)-1] {
		if file.path == "/tmp/virtualapi-start" {
			t.Fatal("server start marker was staged before its configuration, trust, or program")
		}
	}
	if files[len(files)-1].path != "/tmp/virtualapi-start" || string(files[len(files)-1].payload) != "start\n" {
		t.Fatal("server start marker must be staged last after complete file writes")
	}
}

func TestVirtualAPIStartupLogClassifierEmitsOnlyFixedClasses(t *testing.T) {
	for _, tc := range []struct{ log, want string }{
		{"Error: listen EACCES: permission denied", "server-bind-permission-denied"},
		{"Error: listen EADDRINUSE", "server-bind-address-conflict"},
		{"SyntaxError: unexpected token", "server-program-syntax-error"},
		{"invalid exact fixture", "server-rejected-config"},
		{"RangeError: heap out of memory", "server-memory-exhausted"},
		{"synthetic fixture body must never be echoed", "other-startup-error"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
				if len(args) == 0 || args[0] != "logs" {
					t.Fatalf("unexpected diagnostic command %v", args)
				}
				return []byte(tc.log), nil
			}}
			owner := &Owner{Runner: runner}
			got := owner.classifyVirtualAPIStartupLogs(context.Background(), runner, "owned-container")
			if got != tc.want || strings.Contains(got, "fixture body") || len(runner.calls) != 1 {
				t.Fatalf("class=%q want %q calls=%v", got, tc.want, runner.calls)
			}
		})
	}
}

func TestVerifyVirtualAPIIdentityRequiresAllTaskLabels(t *testing.T) {
	labels := map[string]string{
		labelSandboxID:          testSandboxID,
		labelRuntimeID:          "virtualapi:" + "0123456789abcdef01234567",
		labelSandboxOwner:       testSandboxID,
		labelSandboxRunID:       "virtualapi:" + "0123456789abcdef01234567",
		labelSandboxDecl:        "true",
		labelCompanionResource:  virtualAPIContainerResource,
		labelObjectKind:         virtualAPIContainerKind,
		labelHTTPMockConfigHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	runtimeID := labels[labelSandboxRunID]
	var container containerInspect
	container.Config.Labels = labels
	if err := verifyVirtualAPIIdentity(container, testSandboxID, runtimeID, labels[labelHTTPMockConfigHash], []string{"api.notion.com"}); err != nil {
		t.Fatalf("valid task identity rejected: %v", err)
	}
	labels[labelSandboxDecl] = "false"
	if err := verifyVirtualAPIIdentity(container, testSandboxID, runtimeID, labels[labelHTTPMockConfigHash], []string{"api.notion.com"}); !errors.Is(err, ErrForeignObject) {
		t.Fatalf("undeclared companion accepted: %v", err)
	}
}
