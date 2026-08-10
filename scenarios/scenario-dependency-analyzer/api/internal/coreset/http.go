package coreset

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// RegisterHTTPRoutes mounts core-set reporting routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, scenariosDir func() string) {
	api.GET("/core-set", func(c *gin.Context) {
		root := filepath.Dir(scenariosDir())
		if err := ValidateConfiguredTrustedBaseClosure(root); err != nil {
			c.JSON(http.StatusFailedDependency, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, Compute(scenariosDir()))
	})
}
