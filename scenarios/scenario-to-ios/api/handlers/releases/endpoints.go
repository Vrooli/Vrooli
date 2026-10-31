package releases

import "scenario-to-ios/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "ios_matrix", Path: "/api/v1/ios/matrix", Method: "GET", Summary: "Show the iOS validation matrix readiness", Category: "releases", RESTException: module.OpsRESTException("Operator report surface.")},
	{ID: "ios_validation_catalog", Path: "/api/v1/validation/catalog", Method: "GET", Summary: "Show the provider-owned iOS validation catalog", Category: "releases", RESTException: module.OpsRESTException("Catalog is a read-only matrix selection surface.")},
	{ID: "ios_validation_profiles", Path: "/api/v1/validation/profiles", Method: "GET", Summary: "Show validation environment profiles", Category: "releases", RESTException: module.OpsRESTException("Profile catalog is provider-neutral.")},
	{ID: "ios_validation_matrices_list", Path: "/api/v1/validation/matrices", Method: "GET", Summary: "List durable iOS validation matrix runs", Category: "releases", RESTException: module.OpsRESTException("Matrix listing is server-owned and durable.")},
	{ID: "ios_validation_matrices_create", Path: "/api/v1/validation/matrices", Method: "POST", Summary: "Create a durable iOS validation matrix run", Category: "releases", RESTException: module.OpsJSONRESTException("Matrix creation accepts a JSON selection and is server-owned.")},
	{ID: "ios_validation_matrix", Path: "/api/v1/validation/matrices/{run_id}", Method: "GET", Summary: "Inspect a durable iOS validation matrix run", Category: "releases", RESTException: module.OpsRESTException("Run identity and evidence remain producer-owned.")},
	{ID: "ios_validation_matrix_start", Path: "/api/v1/validation/matrices/{run_id}/start", Method: "POST", Summary: "Start a server-owned iOS validation matrix", Category: "releases", RESTException: module.OpsRESTException("Start is asynchronous and durable.")},
	{ID: "ios_validation_matrix_wait", Path: "/api/v1/validation/matrices/{run_id}/wait", Method: "GET", Summary: "Wait for a durable iOS validation matrix", Category: "releases", RESTException: module.OpsRESTException("Wait returns the producer-owned terminal report.")},
	{ID: "ios_validation_matrix_abort", Path: "/api/v1/validation/matrices/{run_id}/abort", Method: "POST", Summary: "Abort a durable iOS validation matrix", Category: "releases", RESTException: module.OpsRESTException("Abort preserves the cancelled run record.")},
	{ID: "ios_validation_matrix_rerun", Path: "/api/v1/validation/matrices/{run_id}/rerun", Method: "POST", Summary: "Rerun selected iOS validation cells", Category: "releases", RESTException: module.OpsJSONRESTException("Rerun creates an independent immutable run.")},
	{ID: "ios_validation_matrix_compare", Path: "/api/v1/validation/matrices/{run_id}/compare/{prior_run_id}", Method: "GET", Summary: "Compare two iOS validation matrix runs", Category: "releases", RESTException: module.OpsRESTException("Comparison does not mutate either run.")},
}
