//go:build journey

package dependencygovernance

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestLauncherGovernanceOwner runs the real governance handler and installer
// in the launcher's isolated HOME. No registry records or install outcomes are
// substituted; the current source registry still governs every acquisition.
func TestLauncherGovernanceOwner(t *testing.T) {
	root := os.Getenv("LAUNCHER_GOVERNANCE_ROOT")
	if root == "" {
		t.Skip("started by the served Web Console full-deployment journey")
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterConnectRoutes(router, func() string { return filepath.Join(root, "scenarios") })
	done := make(chan struct{}, 1)
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) })
	router.GET("/__journey/stop", func(c *gin.Context) { done <- struct{}{}; c.Status(http.StatusNoContent) })
	server := httptest.NewServer(router)
	defer server.Close()
	fmt.Println("LAUNCHER_GOVERNANCE_URL=" + server.URL)
	<-done
}
