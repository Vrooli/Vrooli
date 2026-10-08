package ontology

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestModuleRegistersOntologyConnectHandler(t *testing.T) {
	mod := Module(nil)
	require.Equal(t, "ontology", mod.Name)
	require.Len(t, mod.Endpoints, 15)

	router := mux.NewRouter()
	mod.Mount(router)

	req, err := http.NewRequest(http.MethodPost, "/vrooli.tech_tree_designer.v1.ontology.OntologyService/ListCapabilities", nil)
	require.NoError(t, err)
	match := &mux.RouteMatch{}
	require.True(t, router.Match(req, match))
}
