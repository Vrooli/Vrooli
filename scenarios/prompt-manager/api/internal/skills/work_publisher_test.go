package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPWorkPublisherCreateAndRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"item":{"kind":"execute","name":"promotion-1"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"item":{"kind":"execute","name":"promotion-1","review_status":"approved","rationale":"ship"}}`))
	}))
	defer server.Close()

	publisher := newHTTPWorkPublisher(server.URL)
	ref, err := publisher.CreateWork(context.Background(), "promotion-1", "Promote", "ship")
	if err != nil || ref != "execute/promotion-1" {
		t.Fatalf("CreateWork() = %q, %v", ref, err)
	}
	item, err := publisher.GetWork(context.Background(), ref)
	if err != nil || item.ReviewStatus != "approved" {
		t.Fatalf("GetWork() = %+v, %v", item, err)
	}
}
