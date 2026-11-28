package rehearsals

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var n8nJSONFieldExpression = regexp.MustCompile(`^\s*=\{\{\s*\$json\.([A-Za-z0-9_.]+)\s*}}\s*$`)

// compileTaskHTTPMockWorkflow converts API-described provider nodes in an
// unmodified public workflow to task-local HTTP Request nodes, then binds
// direct HTTP Request nodes to the exact owner-issued mock routes. The source
// export is not modified. The compiler supports trigger-independent native
// provider adaptation and finite exact routing for API-description selectors.
func compileTaskHTTPMockWorkflow(raw string, workflow Workflow, profile HTTPMockProfile, addresses []HTTPMockOperationAddress) (string, error) {
	if err := validateExportedHTTPMockProfile(profile); err != nil {
		return "", errors.New("task workflow compiler requires a validated synthetic HTTP mock profile")
	}
	if len(addresses) != len(profile.Operations) || len(addresses) == 0 {
		return "", errors.New("task workflow compiler requires one owner route address per API operation")
	}
	addressBySelector := make(map[string]HTTPMockOperationAddress, len(addresses))
	for _, address := range addresses {
		key := sandboxOperationSelectorKey(address.Selector)
		if _, exists := addressBySelector[key]; exists {
			return "", errors.New("task workflow compiler received duplicate route selectors")
		}
		addressBySelector[key] = address
	}
	for _, operation := range profile.Operations {
		address, found := addressBySelector[sandboxOperationSelectorKey(operation.Selector)]
		if !found {
			return "", errors.New("task workflow compiler is missing an API operation route address")
		}
		parsed, err := validatedSandboxOperationURL(address)
		if err != nil || !strings.EqualFold(address.Method, operation.Route.Method) || !strings.EqualFold(parsed.Hostname(), operation.Route.Host) || parsed.Path != operation.Route.Path || parsed.RawQuery != operation.Route.Query {
			return "", errors.New("task route address does not match its API-described fixture")
		}
	}

	// First rewrite the workflow's original HTTP Request nodes. Provider nodes
	// are converted after this pass so their already-verified owner URL does not
	// get reinterpreted as an untrusted source URL.
	compiled, err := rewriteWorkflowHTTPRequestsForSandbox(raw, addresses)
	if err != nil {
		return "", err
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(compiled), &document); err != nil || document == nil {
		return "", errors.New("compiled task workflow is not a JSON object")
	}
	nodes, ok := document["nodes"].([]any)
	if !ok {
		return "", errors.New("compiled task workflow has no node list")
	}
	workflowNodes := make(map[string]WorkflowNode, len(workflow.Nodes)*2)
	for _, node := range workflow.Nodes {
		workflowNodes["id:"+node.ID] = node
		workflowNodes["name:"+node.Name] = node
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok {
			return "", errors.New("compiled task workflow contains a non-object node")
		}
		typeName, _ := node["type"].(string)
		if typeName == "n8n-nodes-base.httpRequest" {
			continue
		}
		id, _ := node["id"].(string)
		name, _ := node["name"].(string)
		parsedNode, found := workflowNodes["id:"+id]
		if !found {
			parsedNode, found = workflowNodes["name:"+name]
		}
		if !found || parsedNode.Request == nil || !isNativeServiceNode(typeName) {
			continue
		}
		selected, dynamic, err := selectTaskHTTPMockOperation(node, parsedNode, profile.Operations, addressBySelector)
		if err != nil {
			return "", fmt.Errorf("compile native provider node %q: %w", safeNodeLabel(name, id), err)
		}
		if dynamic != nil {
			parameters := map[string]any{
				"authentication": "none",
				"method":         http.MethodGet,
				"url":            dynamic.URLExpression,
				"options":        map[string]any{},
			}
			node["type"], node["typeVersion"], node["parameters"] = "n8n-nodes-base.httpRequest", 4.2, parameters
			delete(node, "credentials")
			continue
		}
		route := selected.Route
		parameters := map[string]any{
			"authentication": "none", "method": route.Method, "url": selected.Address.URL,
			"options": map[string]any{},
		}
		if route.Method == http.MethodPost || route.Method == http.MethodPut || route.Method == http.MethodPatch {
			parameters["sendBody"] = true
			parameters["specifyBody"] = "json"
			parameters["jsonBody"] = "={{ $json }}"
		}
		node["type"], node["typeVersion"], node["parameters"] = "n8n-nodes-base.httpRequest", 4.2, parameters
		delete(node, "credentials")
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("encode compiled synthetic task workflow")
	}
	return string(encoded), nil
}

type selectedTaskHTTPMockOperation struct {
	Route   HTTPMockRoute
	Address HTTPMockOperationAddress
}

type dynamicTaskHTTPMockOperations struct {
	URLExpression string
}

func selectTaskHTTPMockOperation(node map[string]any, workflowNode WorkflowNode, operations []HTTPMockProfileOperation, addresses map[string]HTTPMockOperationAddress) (selectedTaskHTTPMockOperation, *dynamicTaskHTTPMockOperations, error) {
	request := workflowNode.Request
	var candidates []HTTPMockProfileOperation
	for _, operation := range operations {
		selector := operation.Selector
		if nativeOperationKey(selector.NodeType, selector.Resource, selector.Operation) == nativeOperationKey(request.NodeType, request.Resource, request.Action) {
			candidates = append(candidates, operation)
		}
	}
	if len(candidates) == 0 {
		return selectedTaskHTTPMockOperation{}, nil, errors.New("no API description route matches the provider node")
	}
	parameters, _ := node["parameters"].(map[string]any)
	var exact []HTTPMockProfileOperation
	dynamicKeys := map[string]bool{}
	for _, candidate := range candidates {
		allStatic := true
		for key, expected := range candidate.Selector.InputMatch {
			value, present := n8nStaticParameterValue(parameters, key)
			if !present {
				allStatic = false
				if n8nDynamicParameterPath(parameters, key, workflowNode) != "" {
					dynamicKeys[key] = true
				}
				continue
			}
			if fmt.Sprint(value) != fmt.Sprint(expected) {
				allStatic = false
				break
			}
		}
		if allStatic {
			exact = append(exact, candidate)
		}
	}
	if len(exact) == 1 {
		candidate := exact[0]
		return selectedTaskHTTPMockOperation{Route: candidate.Route, Address: addresses[sandboxOperationSelectorKey(candidate.Selector)]}, nil, nil
	}
	if len(exact) > 1 {
		return selectedTaskHTTPMockOperation{}, nil, errors.New("multiple API routes match the provider node's static parameters")
	}
	if len(dynamicKeys) == 1 {
		var key string
		for candidateKey := range dynamicKeys {
			key = candidateKey
		}
		path := n8nDynamicParameterPath(parameters, key, workflowNode)
		if path == "" {
			return selectedTaskHTTPMockOperation{}, nil, errors.New("API route selector has no safe dynamic input field")
		}
		choices := make(map[string]string, len(candidates))
		for _, candidate := range candidates {
			if len(candidate.Selector.InputMatch) != 1 {
				return selectedTaskHTTPMockOperation{}, nil, errors.New("dynamic route selection supports exactly one selector field")
			}
			value, exists := candidate.Selector.InputMatch[key]
			if !exists {
				return selectedTaskHTTPMockOperation{}, nil, errors.New("dynamic route variants do not share one selector field")
			}
			address := addresses[sandboxOperationSelectorKey(candidate.Selector)]
			if !oneOfFold(candidate.Route.Method, http.MethodGet, http.MethodHead) {
				return selectedTaskHTTPMockOperation{}, nil, errors.New("dynamic selector routing is restricted to read-only methods")
			}
			choices[fmt.Sprint(value)] = address.URL
		}
		choicesJSON, err := json.Marshal(choices)
		if err != nil || len(choicesJSON) > 16<<10 {
			return selectedTaskHTTPMockOperation{}, nil, errors.New("dynamic task route table is invalid or exceeds its bound")
		}
		return selectedTaskHTTPMockOperation{}, &dynamicTaskHTTPMockOperations{URLExpression: "={{ (" + string(choicesJSON) + ")[" + path + "] }}"}, nil
	}
	return selectedTaskHTTPMockOperation{}, nil, errors.New("no exact static or bounded dynamic API selector matches the provider node")
}

func n8nStaticParameterValue(parameters map[string]any, key string) (any, bool) {
	raw, found := parameters[key]
	if !found {
		return nil, false
	}
	switch value := raw.(type) {
	case string, float64, bool:
		return value, true
	case map[string]any:
		if value, exists := value["value"]; exists {
			switch typed := value.(type) {
			case string, float64, bool:
				return typed, true
			}
		}
	}
	return nil, false
}

func n8nDynamicParameterPath(parameters map[string]any, key string, node WorkflowNode) string {
	if node.Type == "n8n-nodes-base.rssFeedRead" && strings.TrimPrefix(node.RSSURLPath, ".") == key && validJSONPath(key) {
		return "$json." + key
	}
	if raw, exists := parameters[key]; exists {
		if expression, ok := raw.(string); ok {
			if match := n8nJSONFieldExpression.FindStringSubmatch(expression); len(match) == 2 && validJSONPath(match[1]) {
				return "$json." + match[1]
			}
		}
	}
	return ""
}

func sandboxOperationSelectorKey(selector HTTPMockProfileSelector) string {
	return nativeOperationKey(selector.NodeType, selector.Resource, selector.Operation) + ":" + canonicalInputMatch(selector.InputMatch)
}

func safeNodeLabel(name, id string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		value = id
	}
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

// HTTPMockOperationAddresses builds deterministic addresses for owner tests
// and task-local native runs that own the fixed 18080 companion. Workspace
// Sandbox callers must instead pass the validated addresses from its receipt.
func HTTPMockOperationAddresses(profile HTTPMockProfile) ([]HTTPMockOperationAddress, error) {
	if err := validateExportedHTTPMockProfile(profile); err != nil {
		return nil, errors.New("cannot compile addresses from an invalid HTTP mock profile")
	}
	addresses := make([]HTTPMockOperationAddress, 0, len(profile.Operations))
	for _, operation := range profile.Operations {
		address, err := syntheticHTTPRouteURL(operation.Route, "http", "18080")
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: operation.Route.Method, URL: address})
	}
	return addresses, nil
}
