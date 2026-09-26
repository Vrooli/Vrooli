package codesigning

import (
	"errors"
	"net/http"
	"testing"
)

func TestHandler_DeletePlatformSigningAndErrorPaths(t *testing.T) {
	h, repo := setupTestHandler(t)
	repo.AddConfig("p1", &SigningConfig{Enabled: true, Linux: &LinuxSigningConfig{GPGKeyID: "ABC"}})
	if rr := makeRequest(t, h.DeletePlatformSigning, http.MethodDelete, "/signing/p1/linux", nil, map[string]string{"id": "p1", "platform": PlatformLinux}); rr.Code != http.StatusOK {
		t.Fatalf("delete platform status=%d body=%s", rr.Code, rr.Body.String())
	}
	for _, tc := range []struct {
		name    string
		vars    map[string]string
		repoErr error
		want    int
	}{
		{"missing id", map[string]string{"platform": PlatformLinux}, nil, http.StatusBadRequest},
		{"invalid platform", map[string]string{"id": "p1", "platform": "freebsd"}, nil, http.StatusBadRequest},
		{"not found", map[string]string{"id": "p1", "platform": PlatformLinux}, ErrProfileNotFound, http.StatusNotFound},
		{"repository error", map[string]string{"id": "p1", "platform": PlatformLinux}, errors.New("database"), http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo.deleteError = tc.repoErr
			rr := makeRequest(t, h.DeletePlatformSigning, http.MethodDelete, "/signing", nil, tc.vars)
			if rr.Code != tc.want {
				t.Fatalf("status=%d body=%s want=%d", rr.Code, rr.Body.String(), tc.want)
			}
			repo.deleteError = nil
		})
	}
}
