package dependencygovernance

import (
	"net/http"
	"time"

	apiServer "github.com/vrooli/api-core/server"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
)

// WithInstallTransport gives the typed install RPC its work budget and a bounded
// response grace. Wrap the raw writer before Gin and the other router middleware.
// The usual route and its middleware still handle every request.
func WithInstallTransport(next http.Handler) (http.Handler, error) {
	install, err := apiServer.WithOperationBudget(next, 3*time.Minute)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == governanceconnect.DependencyGovernanceServiceInstallDependencyProcedure {
			install.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}
