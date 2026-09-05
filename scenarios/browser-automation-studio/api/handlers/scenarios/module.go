// Package scenarios hosts the BAS ScenariosService Connect-RPC handler.
//
// ScenariosService is the discovery surface for sibling scenarios running on
// the same host: agents and the UI call it to list scenarios and resolve a
// scenario's primary HTTP port + base URL without hardcoding ports.
package scenarios

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios/scenariosconnect"
)

// Discovery is the narrow seam the scenarios handler depends on. The
// concrete implementation is the package-level scenarioport helpers; tests
// supply an in-memory fake.
type Discovery interface {
	List(ctx context.Context) ([]scenarioport.ScenarioMetadata, error)
	ResolveURL(ctx context.Context, name string) (url string, info *scenarioport.PortInfo, err error)
	Status(ctx context.Context, name string) (string, error)
}

// Deps wires the scenarios handler. Logger is required; Discovery defaults
// to the package-level scenarioport helpers when nil.
type Deps struct {
	Discovery Discovery
	Logger    *logrus.Logger
}

// Module builds the ScenariosService Connect handler and returns it
// wrapped in a connectx.ServiceMount ready for connectx.RegisterChi.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("scenarios.Module requires Deps.Logger")
	}
	if d.Discovery == nil {
		d.Discovery = defaultDiscovery{}
	}
	path, handler := scenariosconnect.NewScenariosServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

// defaultDiscovery routes through the package-level scenarioport helpers
// so production wiring stays a single call to Module(Deps{Logger: ...}).
type defaultDiscovery struct{}

func (defaultDiscovery) List(ctx context.Context) ([]scenarioport.ScenarioMetadata, error) {
	return scenarioport.ListScenarios(ctx)
}

func (defaultDiscovery) ResolveURL(ctx context.Context, name string) (string, *scenarioport.PortInfo, error) {
	return scenarioport.ResolveURL(ctx, name, "")
}

func (defaultDiscovery) Status(ctx context.Context, name string) (string, error) {
	return scenarioport.GetScenarioStatus(ctx, name)
}
