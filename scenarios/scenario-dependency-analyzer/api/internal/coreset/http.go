package coreset

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterHTTPRoutes mounts core-set reporting routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, scenariosDir func() string) {
	api.GET("/core-set", func(c *gin.Context) {
		c.JSON(http.StatusOK, Compute(scenariosDir()))
	})
}
