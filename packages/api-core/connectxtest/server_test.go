package connectxtest

import (
	"io"
	"net/http"
	"testing"

	"github.com/vrooli/api-core/connectx"
)

func TestStartTestServerMountsService(t *testing.T) {
	server := StartTestServer(t, connectx.ServiceMount{
		Path: "/demo.v1.Notes/",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/demo.v1.Notes/List" {
				t.Fatalf("path = %q", r.URL.Path)
			}
			_, _ = io.WriteString(w, "ok")
		}),
	})

	resp, err := server.Client().Post(server.URL+"/demo.v1.Notes/List", "application/json", nil)
	if err != nil {
		t.Fatalf("post mounted service: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.StatusCode, string(body))
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", string(body))
	}
}

func TestStartTestServerMountsMultipleServices(t *testing.T) {
	server := StartTestServer(t,
		connectx.ServiceMount{
			Path: "/demo.v1.Notes/",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "notes")
			}),
		},
		connectx.ServiceMount{
			Path: "demo.v1.Tasks",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "tasks")
			}),
		},
	)

	for path, want := range map[string]string{
		"/demo.v1.Notes/List": "notes",
		"/demo.v1.Tasks/List": "tasks",
	} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if string(body) != want {
			t.Fatalf("body for %s = %q, want %q", path, string(body), want)
		}
	}
}

func TestStartTestServerCleanupClosesServer(t *testing.T) {
	var serverURL string

	t.Run("server lifecycle", func(t *testing.T) {
		server := StartTestServer(t, connectx.ServiceMount{
			Path: "/demo.v1.Notes/",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "ok")
			}),
		})
		serverURL = server.URL
		resp, err := server.Client().Get(server.URL + "/demo.v1.Notes/List")
		if err != nil {
			t.Fatalf("server should be reachable before cleanup: %v", err)
		}
		_ = resp.Body.Close()
	})

	_, err := http.Get(serverURL + "/demo.v1.Notes/List")
	if err == nil {
		t.Fatal("expected request to fail after t.Cleanup closed the server")
	}
}
