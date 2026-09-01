package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestRoutesHaveNoWritePathOutsideTelemetry(t *testing.T) {
	s := NewServer(testRegistry())
	err := s.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		methods, err := route.GetMethods()
		if err != nil {
			return err
		}
		path, _ := route.GetPathTemplate()
		for _, method := range methods {
			if method != http.MethodGet && !strings.Contains(path, "/debug/") {
				t.Errorf("unexpected %s route %s", method, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
