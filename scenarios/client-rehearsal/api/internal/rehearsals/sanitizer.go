package rehearsals

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const maxWorkflowBytes = 512 * 1024

type rawWorkflow struct {
	ID          string
	Name        string
	Nodes       []any
	Connections any
	Settings    map[string]any
}

// normalizeWorkflow accepts one n8n workflow export or a synthetic bundle of
// exports. The returned graph includes only a narrow inert interpreter
// surface; original workflow JSON, node credentials, request bodies, and
// headers are never returned to the caller or persisted by this domain.
func normalizeWorkflow(raw string) (Workflow, string, uint32, error) {
	if len(raw) == 0 || len(raw) > maxWorkflowBytes {
		return Workflow{}, "", 0, ErrInvalid{Field: "workflow_json", Reason: "must be between 1 byte and 512 KiB"}
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return Workflow{}, "", 0, ErrInvalid{Field: "workflow_json", Reason: "must be valid JSON"}
	}
	replacements := sanitize(document)
	rootDoc := document
	if nested, ok := document["workflow"].(map[string]any); ok {
		rootDoc = nested
	}
	rawDocs, err := unpackWorkflowDocs(rootDoc)
	if err != nil {
		return Workflow{}, "", replacements, err
	}
	workflows := make(map[string]*Workflow, len(rawDocs))
	for index := range rawDocs {
		doc := rawDocs[index]
		if doc.ID == "" {
			if doc.Name != "" {
				doc.ID = doc.Name
			} else {
				doc.ID = fmt.Sprintf("workflow-%d", index+1)
			}
		}
		rawDocs[index].ID = doc.ID
		workflow, err := normalizeN8NWorkflow(doc)
		if err != nil {
			return Workflow{}, "", replacements, err
		}
		if _, exists := workflows[workflow.ID]; exists {
			return Workflow{}, "", replacements, ErrInvalid{Field: "workflows", Reason: "workflow ids must be unique"}
		}
		workflows[workflow.ID] = &workflow
	}
	entryID, _ := rootDoc["entryWorkflowId"].(string)
	if entryID == "" {
		entryID = rawDocs[0].ID
	}
	root := workflows[entryID]
	if root == nil {
		return Workflow{}, "", replacements, ErrInvalid{Field: "entryWorkflowId", Reason: "must identify a workflow in the bundle"}
	}
	for _, workflow := range workflows {
		if workflow.ErrorWorkflowID != "" {
			if workflows[workflow.ErrorWorkflowID] == nil {
				return Workflow{}, "", replacements, ErrInvalid{Field: "settings.errorWorkflow", Reason: "references a workflow missing from the import bundle"}
			}
			workflow.Subworkflows[workflow.ErrorWorkflowID] = workflows[workflow.ErrorWorkflowID]
		}
		for _, node := range workflow.Nodes {
			if node.SubworkflowID == "" {
				continue
			}
			child := workflows[node.SubworkflowID]
			if child == nil {
				return Workflow{}, "", replacements, ErrInvalid{Field: "nodes.parameters.workflowId", Reason: "subworkflow must be included in the import bundle"}
			}
			workflow.Subworkflows[node.SubworkflowID] = child
		}
	}
	fingerprint := workflowStructureFingerprint(root)
	sanitized, err := json.Marshal(fingerprint)
	if err != nil {
		return Workflow{}, "", replacements, fmt.Errorf("encode sanitized workflow structure: %w", err)
	}
	hash := sha256.Sum256(sanitized)
	return *root, hex.EncodeToString(hash[:]), replacements, nil
}

func unpackWorkflowDocs(root map[string]any) ([]rawWorkflow, error) {
	if rawBundles, ok := root["workflows"].([]any); ok {
		if len(rawBundles) == 0 || len(rawBundles) > 64 {
			return nil, ErrInvalid{Field: "workflows", Reason: "bundle must contain 1 to 64 workflows"}
		}
		result := make([]rawWorkflow, 0, len(rawBundles))
		for _, raw := range rawBundles {
			doc, ok := raw.(map[string]any)
			if !ok {
				return nil, ErrInvalid{Field: "workflows", Reason: "each workflow must be an object"}
			}
			parsed, err := rawWorkflowFrom(doc)
			if err != nil {
				return nil, err
			}
			result = append(result, parsed)
		}
		return result, nil
	}
	doc, err := rawWorkflowFrom(root)
	if err != nil {
		return nil, err
	}
	return []rawWorkflow{doc}, nil
}

func rawWorkflowFrom(doc map[string]any) (rawWorkflow, error) {
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 || len(nodes) > 256 {
		return rawWorkflow{}, ErrInvalid{Field: "nodes", Reason: "n8n workflow must contain 1 to 256 nodes"}
	}
	id, _ := doc["id"].(string)
	name, _ := doc["name"].(string)
	settings, _ := doc["settings"].(map[string]any)
	return rawWorkflow{ID: strings.TrimSpace(id), Name: strings.TrimSpace(name), Nodes: nodes, Connections: doc["connections"], Settings: settings}, nil
}

func normalizeN8NWorkflow(raw rawWorkflow) (Workflow, error) {
	workflow := Workflow{ID: raw.ID, Name: raw.Name, Subworkflows: map[string]*Workflow{}}
	if workflow.ID == "" {
		workflow.ID = workflow.Name
	}
	if workflow.ID == "" {
		workflow.ID = "workflow-1"
	}
	byName := make(map[string]string, len(raw.Nodes))
	for index, value := range raw.Nodes {
		node, ok := value.(map[string]any)
		if !ok {
			return Workflow{}, ErrInvalid{Field: "nodes", Reason: "each node must be an object"}
		}
		name, _ := node["name"].(string)
		typeName, _ := node["type"].(string)
		if typeName == "n8n-nodes-base.stickyNote" {
			continue
		}
		if name == "" || typeName == "" {
			return Workflow{}, ErrInvalid{Field: "nodes", Reason: "every n8n node requires name and type"}
		}
		if _, exists := byName[name]; exists {
			return Workflow{}, ErrInvalid{Field: "nodes", Reason: "node names must be unique"}
		}
		id, _ := node["id"].(string)
		if strings.TrimSpace(id) == "" {
			id = fmt.Sprintf("node-%d", index+1)
		}
		if _, exists := byName["id:"+id]; exists {
			return Workflow{}, ErrInvalid{Field: "nodes", Reason: "node ids must be unique"}
		}
		byName[name] = id
		byName["id:"+id] = id
		params, _ := node["parameters"].(map[string]any)
		model, err := normalizeWorkflowNode(id, name, typeName, params)
		if err != nil {
			return Workflow{}, err
		}
		workflow.Nodes = append(workflow.Nodes, model)
		if model.Trigger != "" && model.Trigger != TriggerError && model.Trigger != TriggerSubworkflow {
			workflow.EntryNodeIDs = append(workflow.EntryNodeIDs, id)
			workflow.TriggerKinds = appendUniqueTrigger(workflow.TriggerKinds, model.Trigger)
		}
		if model.Trigger == TriggerError {
			workflow.ErrorNodeID = id
		}
		if model.Trigger == TriggerSubworkflow {
			if workflow.SubworkflowEntryID != "" {
				return Workflow{}, ErrInvalid{Field: "nodes", Reason: "subworkflow export must contain one Execute Workflow Trigger"}
			}
			workflow.SubworkflowEntryID = id
		}
	}
	connections, _ := raw.Connections.(map[string]any)
	for fromName, rawEntry := range connections {
		from := byName[fromName]
		if from == "" {
			return Workflow{}, ErrInvalid{Field: "connections", Reason: "connection source references a missing node"}
		}
		entry, _ := rawEntry.(map[string]any)
		outputs, _ := entry["main"].([]any)
		for outputPort, rawTargets := range outputs {
			targets, _ := rawTargets.([]any)
			for _, rawTarget := range targets {
				link, _ := rawTarget.(map[string]any)
				targetName, _ := link["node"].(string)
				to := byName[targetName]
				if to == "" {
					return Workflow{}, ErrInvalid{Field: "connections", Reason: "connection references a missing node"}
				}
				inputPort := integer(link["index"])
				workflow.Connections = append(workflow.Connections, Connection{From: from, To: to, OutputPort: outputPort, InputPort: inputPort})
			}
		}
	}
	workflow.ErrorWorkflowID = workflowID(raw.Settings["errorWorkflow"])
	if len(workflow.EntryNodeIDs) == 0 && workflow.ErrorNodeID == "" && workflow.SubworkflowEntryID == "" {
		return Workflow{}, ErrInvalid{Field: "connections", Reason: "workflow must have a recognized trigger entry"}
	}
	if err := validateGraph(workflow); err != nil {
		return Workflow{}, err
	}
	return workflow, nil
}

func normalizeWorkflowNode(id, name, typeName string, params map[string]any) (WorkflowNode, error) {
	node := WorkflowNode{ID: id, Name: name, Type: typeName}
	if kind, ok := classifyTrigger(typeName); ok {
		node.Trigger = kind
		return node, nil
	}
	switch typeName {
	case "n8n-nodes-base.httpRequest":
		request, err := normalizeHTTPNode(id, name, params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.Request = &request
	case "n8n-nodes-base.if":
		condition, err := normalizeCondition(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.Condition = &condition
	case "n8n-nodes-base.switch":
		rules, err := normalizeSwitch(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.SwitchRules = rules
	case "n8n-nodes-base.noOp":
		// A no-op has deterministic pass-through semantics.
	case "n8n-nodes-base.set":
		mode, _ := params["mode"].(string)
		if mode != "raw" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.mode", Reason: "Set currently accepts only a raw JSON-path projection"}
		}
		rawOutput, _ := params["jsonOutput"].(string)
		path, ok := parseJSONExpression(rawOutput)
		if !ok {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.jsonOutput", Reason: "raw Set output must be a simple JSON-path expression"}
		}
		node.OutputPath = path
	case "n8n-nodes-base.respondToWebhook", "n8n-nodes-base.stickyNote":
		// Response delivery belongs to the runtime adapter. Sticky note text is
		// intentionally omitted from the normalized graph.
	case "n8n-nodes-base.executeWorkflow":
		node.SubworkflowID = workflowID(params["workflowId"])
		if node.SubworkflowID == "" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.workflowId", Reason: "Execute Workflow requires a static workflow id"}
		}
	default:
		if !isNativeServiceNode(typeName) {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.type", Reason: "unsupported executable node; native service nodes require an n8n-nodes-base type and static resource/operation fields"}
		}
		resource, _ := params["resource"].(string)
		action, _ := params["operation"].(string)
		resource = strings.TrimSpace(resource)
		action = strings.TrimSpace(action)
		if typeName == "n8n-nodes-base.supabase" && resource == "" {
			// Pinned n8n 2.48.0 Supabase.node.ts defaults resource to "row":
			// https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Supabase/Supabase.node.ts
			resource = "row"
		}
		if typeName == "n8n-nodes-base.supabase" && action == "" {
			// Pinned n8n 2.48.0 RowDescription.ts defaults operation to "create":
			// https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Supabase/RowDescription.ts
			action = "create"
		}
		if resource == "" || action == "" || len(resource) > 100 || len(action) > 100 || strings.ContainsAny(resource+action, "\r\n") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "native service nodes require bounded static resource and operation names"}
		}
		node.Request = &Step{ID: id, Operation: typeName + ":" + resource + "." + action, NodeType: typeName, Resource: resource, Action: action}
	}
	return node, nil
}

func isNativeServiceNode(typeName string) bool {
	return strings.HasPrefix(typeName, "n8n-nodes-base.") && !strings.HasSuffix(typeName, "Trigger")
}

func classifyTrigger(typeName string) (TriggerKind, bool) {
	switch typeName {
	case "n8n-nodes-base.manualTrigger":
		return TriggerManual, true
	case "n8n-nodes-base.webhook":
		return TriggerWebhook, true
	case "n8n-nodes-base.scheduleTrigger":
		return TriggerScheduled, true
	case "n8n-nodes-base.errorTrigger":
		return TriggerError, true
	case "n8n-nodes-base.executeWorkflowTrigger":
		return TriggerSubworkflow, true
	case "n8n-nodes-base.chatTrigger", "@n8n/n8n-nodes-langchain.chatTrigger":
		return TriggerChat, true
	default:
		if strings.HasSuffix(typeName, "Trigger") && strings.Contains(typeName, "n8n-nodes-base.") {
			return TriggerPolling, true
		}
		return "", false
	}
}

func appendUniqueTrigger(kinds []TriggerKind, kind TriggerKind) []TriggerKind {
	for _, existing := range kinds {
		if existing == kind {
			return kinds
		}
	}
	return append(kinds, kind)
}

func normalizeHTTPNode(id, name string, params map[string]any) (Step, error) {
	rawURL, _ := params["url"].(string)
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || strings.Contains(rawURL, "={{") {
		return Step{}, ErrInvalid{Field: "nodes.parameters.url", Reason: "HTTP Request nodes require a static absolute HTTP(S) URL; expressions are deferred"}
	}
	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return Step{}, ErrInvalid{Field: "nodes.parameters.method", Reason: "unsupported HTTP method"}
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	return Step{ID: id, Operation: method + " " + path, Method: method, Host: parsed.Hostname(), URL: path}, nil
}

func normalizeCondition(params map[string]any) (Condition, error) {
	conditionsRoot, _ := params["conditions"].(map[string]any)
	conditions, _ := conditionsRoot["conditions"].([]any)
	if len(conditions) == 0 {
		for _, legacy := range []string{"string", "number", "boolean", "dateTime", "array", "object"} {
			if values, ok := conditionsRoot[legacy].([]any); ok && len(values) > 0 {
				conditions = values
				break
			}
		}
	}
	if len(conditions) != 1 {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "first graph adapter supports exactly one condition per If node"}
	}
	first, _ := conditions[0].(map[string]any)
	return conditionFromEntry(first)
}

func conditionFromEntry(entry map[string]any) (Condition, error) {
	left, _ := entry["leftValue"].(string)
	if left == "" {
		left, _ = entry["value1"].(string)
	}
	path, ok := parseJSONExpression(left)
	if !ok {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "condition must use a simple JSON path expression"}
	}
	operatorRoot, _ := entry["operator"].(map[string]any)
	operator, _ := operatorRoot["operation"].(string)
	if operator == "" {
		operator, _ = entry["operation"].(string)
	}
	if operator == "" {
		operator = "equals"
	}
	switch operator {
	case "equals", "notEquals", "contains", "gt", "gte", "lt", "lte":
	default:
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.operator", Reason: "operator must be equals, notEquals, contains, gt, gte, lt, or lte"}
	}
	value, ok := entry["rightValue"]
	if !ok {
		value = entry["value2"]
	}
	return Condition{Path: path, Operator: operator, Value: value}, nil
}

func normalizeSwitch(params map[string]any) ([]SwitchRule, error) {
	rulesRoot, _ := params["rules"].(map[string]any)
	values, _ := rulesRoot["values"].([]any)
	if len(values) == 0 {
		values, _ = rulesRoot["rules"].([]any)
	}
	if len(values) == 0 || len(values) > 32 {
		return nil, ErrInvalid{Field: "nodes.parameters.rules", Reason: "Switch requires 1 to 32 rules"}
	}
	legacyPath := ""
	if raw, _ := params["value1"].(string); raw != "" {
		var ok bool
		legacyPath, ok = parseJSONExpression(raw)
		if !ok {
			return nil, ErrInvalid{Field: "nodes.parameters.value1", Reason: "Switch value1 must be a simple JSON path expression"}
		}
	}
	result := make([]SwitchRule, 0, len(values))
	for index, rawRule := range values {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			return nil, ErrInvalid{Field: "nodes.parameters.rules", Reason: "each Switch rule must be an object"}
		}
		outputPort := index
		if explicit, ok := rule["output"].(float64); ok {
			outputPort = int(explicit)
		}
		condition, err := switchCondition(rule, legacyPath)
		if err != nil {
			return nil, err
		}
		result = append(result, SwitchRule{Condition: condition, OutputPort: outputPort})
	}
	return result, nil
}

func switchCondition(rule map[string]any, legacyPath string) (Condition, error) {
	conditionsRoot, _ := rule["conditions"].(map[string]any)
	conditions, _ := conditionsRoot["conditions"].([]any)
	if len(conditions) > 0 {
		if len(conditions) != 1 {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.rules.conditions", Reason: "each Switch rule supports exactly one condition"}
		}
		entry, _ := conditions[0].(map[string]any)
		return conditionFromEntry(entry)
	}
	if legacyPath == "" {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.rules", Reason: "legacy Switch rules require a simple value1 expression"}
	}
	return Condition{Path: legacyPath, Operator: "equals", Value: rule["value2"]}, nil
}

func parseJSONExpression(raw string) (string, bool) {
	expression := strings.TrimSpace(raw)
	if strings.HasPrefix(expression, "=") {
		expression = strings.TrimSpace(strings.TrimPrefix(expression, "="))
	}
	if !strings.HasPrefix(expression, "{{") || !strings.HasSuffix(expression, "}}") {
		return "", false
	}
	expression = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "{{"), "}}"))
	path := ""
	if strings.HasPrefix(expression, "$json.") {
		path = "." + strings.TrimPrefix(expression, "$json.")
	} else if strings.HasPrefix(expression, "$node[") {
		jsonIndex := strings.Index(expression, "].json")
		if jsonIndex < 0 {
			return "", false
		}
		path = expression[jsonIndex+len("].json"):]
		if strings.HasPrefix(path, ".") {
			path = strings.TrimPrefix(path, ".")
		}
	} else {
		return "", false
	}
	parts := []string{}
	for len(path) > 0 {
		if strings.HasPrefix(path, ".") {
			path = path[1:]
			end := strings.IndexAny(path, ".[ ")
			part := path
			if end >= 0 {
				part = path[:end]
			}
			if !validPathPart(part) {
				return "", false
			}
			parts = append(parts, part)
			path = path[len(part):]
			continue
		}
		if strings.HasPrefix(path, "[\"") {
			end := strings.Index(path[2:], "\"]")
			if end < 0 {
				return "", false
			}
			part := path[2 : 2+end]
			if !validPathPart(part) {
				return "", false
			}
			parts = append(parts, part)
			path = path[2+end+2:]
			continue
		}
		return "", false
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "."), true
}

func validPathPart(part string) bool {
	if part == "" {
		return false
	}
	for _, char := range part {
		if !(char == '_' || char == '-' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

func validateGraph(workflow Workflow) error {
	nodes := make(map[string]WorkflowNode, len(workflow.Nodes))
	indegree := make(map[string]int, len(workflow.Nodes))
	outgoing := make(map[string][]string, len(workflow.Nodes))
	for _, node := range workflow.Nodes {
		nodes[node.ID] = node
	}
	for _, edge := range workflow.Connections {
		if edge.OutputPort < 0 || edge.OutputPort > 32 || edge.InputPort < 0 || edge.InputPort > 32 {
			return ErrInvalid{Field: "connections", Reason: "ports must be between 0 and 32"}
		}
		indegree[edge.To]++
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
	}
	queue := make([]string, 0, len(workflow.Nodes))
	for _, node := range workflow.Nodes {
		if indegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range outgoing[id] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if visited != len(nodes) {
		return ErrInvalid{Field: "connections", Reason: "cycles and loop nodes are not supported by this deterministic graph adapter"}
	}
	for _, entry := range workflow.EntryNodeIDs {
		if nodes[entry].ID == "" {
			return ErrInvalid{Field: "connections", Reason: "trigger entry references a missing node"}
		}
	}
	if workflow.ErrorNodeID != "" && nodes[workflow.ErrorNodeID].ID == "" {
		return ErrInvalid{Field: "nodes", Reason: "Error Trigger entry references a missing node"}
	}
	if workflow.SubworkflowEntryID != "" && nodes[workflow.SubworkflowEntryID].ID == "" {
		return ErrInvalid{Field: "nodes", Reason: "Execute Workflow Trigger entry references a missing node"}
	}
	starts := append([]string(nil), workflow.EntryNodeIDs...)
	if workflow.ErrorNodeID != "" {
		starts = append(starts, workflow.ErrorNodeID)
	}
	if workflow.SubworkflowEntryID != "" {
		starts = append(starts, workflow.SubworkflowEntryID)
	}
	reachable := map[string]bool{}
	for _, start := range starts {
		queue := []string{start}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if reachable[id] {
				continue
			}
			reachable[id] = true
			queue = append(queue, outgoing[id]...)
		}
	}
	if len(reachable) != len(nodes) {
		return ErrInvalid{Field: "connections", Reason: "every node must be reachable from a workflow trigger"}
	}
	return nil
}

func workflowID(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if id, ok := typed["value"].(string); ok {
			return strings.TrimSpace(id)
		}
		if id, ok := typed["id"].(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func integer(value any) int {
	if number, ok := value.(float64); ok {
		return int(number)
	}
	return 0
}

func workflowStructureFingerprint(root *Workflow) any {
	visited := map[string]bool{}
	var project func(*Workflow) any
	project = func(workflow *Workflow) any {
		if workflow == nil || visited[workflow.ID] {
			return nil
		}
		visited[workflow.ID] = true
		nodes := make([]map[string]any, 0, len(workflow.Nodes))
		for _, node := range workflow.Nodes {
			entry := map[string]any{"type": node.Type, "trigger": node.Trigger}
			if node.Request != nil {
				entry["request"] = []string{node.Request.Method, node.Request.Host, node.Request.URL, node.Request.NodeType, node.Request.Resource, node.Request.Action}
			}
			if node.OutputPath != "" {
				entry["output_path"] = node.OutputPath
			}
			if node.SubworkflowID != "" {
				entry["subworkflow"] = node.SubworkflowID
			}
			nodes = append(nodes, entry)
		}
		edges := append([]Connection(nil), workflow.Connections...)
		sort.Slice(edges, func(i, j int) bool {
			if edges[i].From == edges[j].From {
				if edges[i].OutputPort == edges[j].OutputPort {
					return edges[i].To < edges[j].To
				}
				return edges[i].OutputPort < edges[j].OutputPort
			}
			return edges[i].From < edges[j].From
		})
		children := make([]any, 0, len(workflow.Subworkflows))
		keys := make([]string, 0, len(workflow.Subworkflows))
		for id := range workflow.Subworkflows {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			children = append(children, project(workflow.Subworkflows[id]))
		}
		return map[string]any{"nodes": nodes, "edges": edges, "triggers": workflow.TriggerKinds, "error": workflow.ErrorWorkflowID, "children": children}
	}
	return project(root)
}

func safeSourceURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "redacted://invalid"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.Scheme + "://" + parsed.Host + parsed.EscapedPath()
}

func sanitize(value any) uint32 {
	var count uint32
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if secretKey(key) {
				if nonEmpty(child) {
					count++
				}
				v[key] = "synthetic://credential-replaced"
				continue
			}
			count += sanitize(child)
		}
	case []any:
		for _, child := range v {
			count += sanitize(child)
		}
	}
	return count
}

func secretKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, part := range []string{"credential", "secret", "token", "password", "apikey", "privatekey", "authorization", "clientid"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func nonEmpty(value any) bool { return value != nil && value != "" }
