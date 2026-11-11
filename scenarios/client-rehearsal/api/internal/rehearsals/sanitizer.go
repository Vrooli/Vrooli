package rehearsals

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
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
		typeVersion, _ := node["typeVersion"].(float64)
		model, err := normalizeWorkflowNodeVersion(id, name, typeName, params, typeVersion)
		if err != nil {
			return Workflow{}, err
		}
		switch onError, _ := node["onError"].(string); onError {
		case "", "continueRegularOutput":
		case "continueErrorOutput":
			model.ErrorOutput = true
		default:
			return Workflow{}, ErrInvalid{Field: "nodes.onError", Reason: "only continueRegularOutput and continueErrorOutput are modeled"}
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
	return normalizeWorkflowNodeVersion(id, name, typeName, params, 0)
}

func normalizeWorkflowNodeVersion(id, name, typeName string, params map[string]any, typeVersion float64) (WorkflowNode, error) {
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
	case "n8n-nodes-base.filter":
		condition, err := normalizeFilter(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.FilterCondition = &condition
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
		if mode == "" && params["assignments"] != nil {
			mode = "manual"
		}
		if mode == "raw" {
			rawOutput, _ := params["jsonOutput"].(string)
			path, ok := parseJSONExpression(rawOutput)
			if !ok {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.jsonOutput", Reason: "raw Set output must be a simple JSON-path expression"}
			}
			node.OutputPath = path
		} else if mode == "manual" {
			values, projections, expressions, includeInput, err := normalizeSetAssignments(params)
			if err != nil {
				return WorkflowNode{}, err
			}
			node.SetValues = values
			node.SetProjections = projections
			node.SetExpressions = expressions
			node.SetIncludeInput = includeInput
		} else {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.mode", Reason: "Set accepts only bounded manual assignments or a raw JSON-path projection"}
		}
	case "n8n-nodes-base.splitOut":
		field, _ := params["fieldToSplitOut"].(string)
		field = strings.TrimSpace(field)
		if !validJSONPath(field) {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.fieldToSplitOut", Reason: "Split Out requires a bounded static JSON path"}
		}
		include, _ := params["include"].(string)
		if include != "" && include != "noOtherFields" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.include", Reason: "Split Out currently accepts only noOtherFields"}
		}
		node.SplitField = field
	case "n8n-nodes-base.code":
		if !onlyKeys(params, "jsCode", "functionCode", "mode", "language") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Code accepts only source, mode, and language selectors"}
		}
		mode, _ := params["mode"].(string)
		if mode != "" && mode != "runOnceForAllItems" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.mode", Reason: "safe Code supports only bounded runOnceForAllItems mode"}
		}
		language, _ := params["language"].(string)
		if language != "" && language != "javaScript" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.language", Reason: "safe Code interprets only the allowlisted JavaScript subset"}
		}
		source, _ := params["jsCode"].(string)
		functionSource, _ := params["functionCode"].(string)
		if source == "" {
			source = functionSource
		} else if functionSource != "" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "only one Code source field may be provided"}
		}
		if _, err := parseSafeCode(source); err != nil {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.jsCode", Reason: "Code node is outside the bounded safe subset: " + err.Error()}
		}
		node.SafeCode = source
	case "n8n-nodes-base.rssFeedRead":
		if typeVersion != 0 && typeVersion > 1.2 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.typeVersion", Reason: "RSS Feed Read supports only pinned versions through 1.2"}
		}
		if !onlyKeys(params, "url", "options") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "RSS Feed Read accepts only a bounded URL expression and empty options"}
		}
		if options, ok := params["options"].(map[string]any); ok && len(options) != 0 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "RSS Feed Read options are unsupported in synthetic mode"}
		}
		rawURL, _ := params["url"].(string)
		path, ok := parseJSONExpression(rawURL)
		if !ok || strings.Contains(rawURL, "$node[") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.url", Reason: "RSS Feed Read requires a simple current-item JSON path; external URLs are never fetched"}
		}
		node.RSSURLPath = path
	case "n8n-nodes-base.aggregate":
		fields, keepMissing, mergeLists, disableDotNotation, err := normalizeAggregate(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.AggregateFields = fields
		node.AggregateKeepMissing = keepMissing
		node.AggregateMergeLists = mergeLists
		node.AggregateDisableDotNotation = disableDotNotation
	case "n8n-nodes-base.splitInBatches":
		if typeVersion != 3 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.typeVersion", Reason: "Loop Over Items currently supports pinned Split In Batches version 3 only"}
		}
		if !onlyKeys(params, "options") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Loop Over Items accepts only bounded batch options"}
		}
		options, _ := params["options"].(map[string]any)
		if !onlyKeys(options, "batchSize") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "Loop Over Items accepts only a static batchSize"}
		}
		batchSize := 1.0
		if raw, ok := options["batchSize"]; ok {
			var valid bool
			batchSize, valid = raw.(float64)
			if !valid || batchSize < 1 || batchSize > maxSyntheticItems || batchSize != float64(uint32(batchSize)) {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.batchSize", Reason: "batchSize must be a static integer within the synthetic item bound"}
			}
		}
		node.LoopBatchSize = uint32(batchSize)
	case "n8n-nodes-base.wait":
		seconds, amountPath, unit, err := normalizeWaitNode(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.WaitSeconds = seconds
		node.WaitAmountPath = amountPath
		node.WaitUnit = unit
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
			return WorkflowNode{}, ErrInvalid{Field: "nodes.type", Reason: "unsupported executable node; external operations require an allowlisted node type and static resource/operation fields"}
		}
		resource, _ := params["resource"].(string)
		action, _ := params["operation"].(string)
		resource = strings.TrimSpace(resource)
		action = strings.TrimSpace(action)
		if typeName == "n8n-nodes-base.hubspot" && resource == "" && typeVersion == 2 {
			// HubSpot V2's pinned ContactDescription defaults resource to contact.
			// https://raw.githubusercontent.com/n8n-io/n8n/n8n%402.48.0/packages/nodes-base/nodes/Hubspot/V2/ContactDescription.ts
			resource = "contact"
		}
		if typeName == "n8n-nodes-base.notion" && typeVersion == 2 && resource == "databasePage" && action == "" {
			// Pinned Notion v2.48.0 declares databasePage.operation's default as create.
			// https://raw.githubusercontent.com/n8n-io/n8n/n8n%402.48.0/packages/nodes-base/nodes/Notion/shared/descriptions/DatabasePageDescription.ts
			action = "create"
		}
		if typeName == "n8n-nodes-base.notion" && typeVersion == 2 && resource == "" {
			// Notion v2.48.0 selects `page` when the resource field is omitted.
			// The operation, when absent too, is selected below only for explicit node pairs.
			resource = "page"
		}
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
		// n8n exports can omit the UI's selected defaults. Resolve only these
		// stable native operation defaults; the simulator still requires an
		// explicit profile mapping and never performs the service effect.
		if typeName == "n8n-nodes-base.googleSheets" && resource == "" {
			resource = "sheet"
		}
		if typeName == "n8n-nodes-base.googleSheets" && resource == "sheet" && action == "" && typeVersion == 4.6 {
			// Pinned n8n 2.48.0 GoogleSheetsV2 uses Sheet.resource's default
			// operation "read" for node version 4.6.
			action = "read"
		}
		if typeName == "n8n-nodes-base.gmail" {
			if resource == "" {
				resource = "message"
			}
			if action == "" {
				action = "send"
			}
		}
		if typeName == "n8n-nodes-base.github" {
			if resource == "" {
				resource = "issue"
			}
			if action == "" {
				action = "create"
			}
		}
		if typeName == "n8n-nodes-base.slack" && resource == "" && action == "" {
			selectTo, _ := params["select"].(string)
			text, _ := params["text"].(string)
			if (selectTo == "channel" || selectTo == "user") && text != "" && len(text) <= 4096 && !strings.ContainsRune(text, '\x00') {
				// Slack V2 omits unchanged UI defaults in exports. A static destination
				// selector and message text bind the generic Message > Send operation.
				resource, action = "message", "post"
			}
		}
		if typeName == "@n8n/n8n-nodes-langchain.openAi" {
			if resource == "" && action == "" {
				model, _ := params["modelId"].(string)
				messages, _ := params["messages"].(map[string]any)
				values, _ := messages["values"].([]any)
				if model == "" || len(model) > 100 || strings.ContainsAny(model, "\x00\r\n") || strings.Contains(model, "={{") || strings.Contains(model, "{{") || len(values) == 0 || len(values) > 16 {
					return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "OpenAI chat simulation requires a bounded model selector and 1 to 16 message descriptors"}
				}
				for _, rawMessage := range values {
					message, ok := rawMessage.(map[string]any)
					rawRole, rolePresent := message["role"]
					role, roleIsString := rawRole.(string)
					content, contentOK := message["content"].(string)
					if !ok || !contentOK || (rolePresent && !roleIsString) || (rolePresent && role != "system" && role != "user" && role != "assistant") || len(content) > 4096 {
						return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.messages", Reason: "OpenAI simulation supports only bounded static role/content descriptors"}
					}
				}
				resource, action = "chat", "completion"
			} else if resource != "chat" || action != "completion" {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "the LangChain OpenAI adapter supports only chat completion through a synthetic profile"}
			}
		}
		if typeName == "n8n-nodes-base.postgres" {
			if resource == "" {
				resource = "query"
			}
			if action == "executeQuery" {
				query, _ := params["query"].(string)
				statement, ok := classifyPostgresStatement(query)
				if !ok {
					return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.query", Reason: "Postgres simulation accepts one bounded static SELECT, INSERT, UPDATE, or DELETE statement"}
				}
				// Retain only the statement class for OpenAPI profile routing. SQL,
				// query parameters, and expressions never enter the normalized graph.
				action += "." + statement
			}
		}
		if resource == "" || action == "" || len(resource) > 100 || len(action) > 100 || strings.ContainsAny(resource+action, "\r\n") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: fmt.Sprintf("native service node %q (%s) requires a supported bounded static resource and operation (resource=%q operation=%q)", name, typeName, resource, action)}
		}
		request := &Step{ID: id, Operation: typeName + ":" + resource + "." + action, NodeType: typeName, Resource: resource, Action: action}
		if typeName == "n8n-nodes-base.googleSheets" && resource == "sheet" && action == "read" {
			if locator, ok := params["sheetName"].(map[string]any); ok {
				if value, ok := locator["value"].(string); ok && value != "" && len(value) <= 128 && !strings.ContainsAny(value, "\x00\r\n") && !strings.Contains(value, "={{") {
					request.Input = map[string]any{"sheetName": value}
				}
			}
			node.RequestReplacesInput = true
		}
		node.Request = request
	}
	return node, nil
}

func normalizeSetAssignments(params map[string]any) (map[string]any, map[string]string, map[string]string, bool, error) {
	container, _ := params["assignments"].(map[string]any)
	items, _ := container["assignments"].([]any)
	if len(items) == 0 || len(items) > 64 {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "manual Set requires 1 to 64 bounded assignments"}
	}
	values := make(map[string]any, len(items))
	projections := make(map[string]string)
	expressions := make(map[string]string)
	seen := map[string]bool{}
	for _, item := range items {
		assignment, ok := item.(map[string]any)
		if !ok {
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "each Set assignment must be an object"}
		}
		name, _ := assignment["name"].(string)
		kind, _ := assignment["type"].(string)
		value := assignment["value"]
		if name == "" || len(name) > 100 || strings.ContainsAny(name, "\x00\r\n") || seen[name] {
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set assignment names must be unique, nonempty and bounded"}
		}
		seen[name] = true
		if exprText, ok := value.(string); ok && strings.Contains(exprText, "={{") {
			expression, err := normalizeSafeExpression(exprText)
			if err != nil {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set expression is outside the bounded safe subset: " + err.Error()}
			}
			if path, simple := parseJSONExpression(exprText); simple {
				projections[name] = path
			} else {
				expressions[name] = expression
			}
			continue
		}
		switch kind {
		case "string":
			text, ok := value.(string)
			if !ok || len(text) > 4096 || strings.ContainsAny(text, "\x00") {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set string values must be bounded literals or safe expressions"}
			}
			values[name] = text
		case "number":
			if _, ok := value.(float64); !ok {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set number values must be numeric literals or safe expressions"}
			}
			values[name] = value
		case "boolean":
			if _, ok := value.(bool); !ok {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set boolean values must be boolean literals or safe expressions"}
			}
			values[name] = value
		default:
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set supports bounded string, number and boolean values"}
		}
	}
	includeInput, _ := params["includeOtherFields"].(bool)
	return values, projections, expressions, includeInput, nil
}

func normalizeAggregate(params map[string]any) ([]AggregateField, bool, bool, bool, error) {
	mode, _ := params["aggregate"].(string)
	if mode != "" && mode != "aggregateIndividualFields" {
		return nil, false, false, false, ErrInvalid{Field: "nodes.parameters.aggregate", Reason: "Aggregate supports only bounded Individual Fields mode"}
	}
	container, _ := params["fieldsToAggregate"].(map[string]any)
	rawFields, _ := container["fieldToAggregate"].([]any)
	if len(rawFields) == 0 || len(rawFields) > 64 {
		return nil, false, false, false, ErrInvalid{Field: "nodes.parameters.fieldsToAggregate", Reason: "Aggregate requires 1 to 64 fields"}
	}
	options, _ := params["options"].(map[string]any)
	keepMissing, _ := options["keepMissing"].(bool)
	mergeLists, _ := options["mergeLists"].(bool)
	disableDotNotation, _ := options["disableDotNotation"].(bool)
	fields := make([]AggregateField, 0, len(rawFields))
	seen := map[string]bool{}
	for _, raw := range rawFields {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false, false, false, ErrInvalid{Field: "nodes.parameters.fieldsToAggregate", Reason: "each aggregate field must be an object"}
		}
		path, _ := item["fieldToAggregate"].(string)
		path = strings.TrimSpace(path)
		if !validJSONPath(path) {
			return nil, false, false, false, ErrInvalid{Field: "nodes.parameters.fieldsToAggregate", Reason: "field paths must be bounded static JSON paths"}
		}
		rename, _ := item["renameField"].(bool)
		out, _ := item["outputFieldName"].(string)
		if rename {
			if out == "" || len(out) > 64 || !validPathPart(out) {
				return nil, false, false, false, ErrInvalid{Field: "nodes.parameters.fieldsToAggregate", Reason: "renamed aggregate fields require a bounded static output name"}
			}
		}
		if out == "" {
			out = path
			if index := strings.LastIndex(out, "."); index >= 0 {
				out = out[index+1:]
			}
			if bracket := strings.Index(out, "["); bracket >= 0 {
				out = out[:bracket]
			}
		}
		if out == "" || seen[out] {
			return nil, false, false, false, ErrInvalid{Field: "nodes.parameters.fieldsToAggregate", Reason: "aggregate output names must be nonempty and unique"}
		}
		seen[out] = true
		fields = append(fields, AggregateField{InputPath: path, OutputName: out})
	}
	return fields, keepMissing, mergeLists, disableDotNotation, nil
}

func validJSONPath(path string) bool {
	_, ok := parseJSONPath(path)
	return ok
}

type jsonPathSegment struct {
	key     string
	index   int
	isIndex bool
}

// parseJSONPath accepts a short dotted object path with bounded numeric array
// indexes. It is deliberately not a JavaScript expression parser.
func parseJSONPath(path string) ([]jsonPathSegment, bool) {
	if path == "" || len(path) > 256 {
		return nil, false
	}
	var result []jsonPathSegment
	for pos := 0; pos < len(path); {
		if path[pos] == '.' {
			pos++
			if pos >= len(path) {
				return nil, false
			}
		}
		if path[pos] == '[' {
			end := strings.IndexByte(path[pos:], ']')
			if end < 0 {
				return nil, false
			}
			end += pos
			value := path[pos+1 : end]
			if value == "" || len(value) > 3 {
				return nil, false
			}
			index, err := strconv.Atoi(value)
			if err != nil || index < 0 || index > 100 {
				return nil, false
			}
			result = append(result, jsonPathSegment{index: index, isIndex: true})
			pos = end + 1
		} else {
			start := pos
			for pos < len(path) && path[pos] != '.' && path[pos] != '[' {
				pos++
			}
			key := path[start:pos]
			if !validPathPart(key) || len(key) > 64 {
				return nil, false
			}
			result = append(result, jsonPathSegment{key: key})
		}
		if len(result) > 8 {
			return nil, false
		}
	}
	return result, len(result) > 0
}

func classifyPostgresStatement(query string) (string, bool) {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 16*1024 || strings.ContainsAny(query, "\x00") || strings.Contains(query, "={{") {
		return "", false
	}
	if strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	// This does not parse or execute SQL. Reject multiple statements and retain
	// only a recognized first keyword as a simulation selector.
	if strings.Contains(query, ";") {
		return "", false
	}
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return "", false
	}
	verb := strings.ToLower(fields[0])
	switch verb {
	case "select", "insert", "update", "delete":
		return verb, true
	default:
		return "", false
	}
}

func isNativeServiceNode(typeName string) bool {
	return (strings.HasPrefix(typeName, "n8n-nodes-base.") && !strings.HasSuffix(typeName, "Trigger")) || typeName == "@n8n/n8n-nodes-langchain.openAi"
}

func normalizeWait(params map[string]any) (uint32, error) {
	seconds, amountPath, _, err := normalizeWaitNode(params)
	if err != nil {
		return 0, err
	}
	if amountPath != "" {
		return 0, ErrInvalid{Field: "nodes.parameters.amount", Reason: "dynamic Wait amounts cannot be normalized without an input item"}
	}
	return seconds, nil
}

func normalizeWaitNode(params map[string]any) (uint32, string, string, error) {
	for key := range params {
		if key != "amount" && key != "unit" {
			return 0, "", "", ErrInvalid{Field: "nodes.parameters", Reason: "Wait accepts only bounded static amount and unit fields"}
		}
	}
	unit, _ := params["unit"].(string)
	if unit == "" {
		// Pinned n8n 2.48.0 Wait.node.ts defaults the unit to hours.
		unit = "hours"
	}
	multiplier := uint64(0)
	switch unit {
	case "seconds":
		multiplier = 1
	case "minutes":
		multiplier = 60
	case "hours":
		multiplier = 60 * 60
	case "days":
		multiplier = 24 * 60 * 60
	default:
		return 0, "", "", ErrInvalid{Field: "nodes.parameters.unit", Reason: "Wait supports only static seconds, minutes, hours, or days"}
	}
	amountValue := params["amount"]
	if amount, ok := amountValue.(string); ok {
		if !isCurrentItemJSONExpression(amount) {
			return 0, "", "", ErrInvalid{Field: "nodes.parameters.amount", Reason: "dynamic Wait amount requires a current-item $json path"}
		}
		path, valid := parseJSONExpression(amount)
		if !valid || !validJSONPath(path) {
			return 0, "", "", ErrInvalid{Field: "nodes.parameters.amount", Reason: "dynamic Wait amount requires a simple current-item JSON path"}
		}
		return 0, path, unit, nil
	}
	amount, ok := amountValue.(float64)
	if !ok || amount < 1 {
		return 0, "", "", ErrInvalid{Field: "nodes.parameters.amount", Reason: "Wait amount must be a positive static integer or a simple current-item JSON path"}
	}
	maxSeconds := uint64(30 * 24 * 60 * 60)
	if amount > float64(maxSeconds/multiplier) {
		return 0, "", "", ErrInvalid{Field: "nodes.parameters.amount", Reason: "Wait virtual delay exceeds the 30-day bound"}
	}
	if amount != float64(uint64(amount)) {
		return 0, "", "", ErrInvalid{Field: "nodes.parameters.amount", Reason: "Wait amount must be a positive integer"}
	}
	seconds := uint64(amount) * multiplier
	return uint32(seconds), "", unit, nil
}

func isCurrentItemJSONExpression(raw string) bool {
	expression := strings.TrimSpace(raw)
	if strings.HasPrefix(expression, "=") {
		expression = strings.TrimSpace(strings.TrimPrefix(expression, "="))
	}
	if !strings.HasPrefix(expression, "{{") || !strings.HasSuffix(expression, "}}") {
		return false
	}
	expression = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "{{"), "}}"))
	return strings.HasPrefix(expression, "$json.")
}

func waitUnitSeconds(unit string) uint64 {
	switch unit {
	case "seconds":
		return 1
	case "minutes":
		return 60
	case "hours":
		return 60 * 60
	case "days":
		return 24 * 60 * 60
	default:
		return 0
	}
}

func classifyTrigger(typeName string) (TriggerKind, bool) {
	switch typeName {
	case "n8n-nodes-base.manualTrigger":
		return TriggerManual, true
	case "n8n-nodes-base.cron":
		return TriggerScheduled, true
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

func normalizeFilter(params map[string]any) (Condition, error) {
	if !onlyKeys(params, "conditions", "options") {
		return Condition{}, ErrInvalid{Field: "nodes.parameters", Reason: "Filter accepts only conditions and options"}
	}
	if options, ok := params["options"].(map[string]any); !ok || len(options) != 0 {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "Filter options must be empty"}
	}
	root, ok := params["conditions"].(map[string]any)
	if !ok || !onlyKeys(root, "combinator", "conditions", "options") {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "Filter requires a bounded single-condition object"}
	}
	combinator, _ := root["combinator"].(string)
	if combinator != "and" && combinator != "or" {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.combinator", Reason: "Filter combinator must be and or or"}
	}
	rootOptions, ok := root["options"].(map[string]any)
	if !ok || !onlyKeys(rootOptions, "caseSensitive", "leftValue", "typeValidation") {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.options", Reason: "Filter condition options are unsupported"}
	}
	if rootOptions["caseSensitive"] != true || rootOptions["leftValue"] != "" || (rootOptions["typeValidation"] != "strict" && rootOptions["typeValidation"] != "loose") {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.options", Reason: "Filter options must use the supported case and type settings"}
	}
	conditions, ok := root["conditions"].([]any)
	if !ok || len(conditions) != 1 {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.conditions", Reason: "Filter supports exactly one condition"}
	}
	entry, ok := conditions[0].(map[string]any)
	if !ok || !onlyKeys(entry, "id", "leftValue", "operator", "rightValue") {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.conditions", Reason: "Filter condition fields are unsupported"}
	}
	operator, ok := entry["operator"].(map[string]any)
	if !ok || !onlyKeys(operator, "type", "operation", "singleValue") || operator["type"] != "boolean" || operator["singleValue"] != true {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter supports only single-value boolean conditions"}
	}
	condition, err := conditionFromEntry(entry)
	if err != nil {
		return Condition{}, err
	}
	if condition.Operator != "true" && condition.Operator != "false" {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter supports only boolean true/false predicates"}
	}
	if condition.Operator == "true" && entry["rightValue"] != true || condition.Operator == "false" && entry["rightValue"] != false {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.conditions.rightValue", Reason: "Filter boolean predicate must use its matching boolean value"}
	}
	return condition, nil
}

func onlyKeys(values map[string]any, allowed ...string) bool {
	if values == nil {
		return false
	}
	set := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		set[key] = true
	}
	for key := range values {
		if !set[key] {
			return false
		}
	}
	return true
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
	case "equals", "notEquals", "contains", "gt", "gte", "lt", "lte", "true", "false":
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
		if strings.HasPrefix(path, `["`) {
			end := strings.Index(path[2:], `"]`)
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
		if strings.HasPrefix(path, "[") {
			end := strings.IndexByte(path, ']')
			if end < 0 || len(parts) == 0 {
				return "", false
			}
			rawIndex := path[1:end]
			index, err := strconv.Atoi(rawIndex)
			if err != nil || index < 0 || index > 100 {
				return "", false
			}
			parts[len(parts)-1] += fmt.Sprintf("[%d]", index)
			path = path[end+1:]
			continue
		}
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
	feedbackEdges, err := loopFeedbackEdges(workflow)
	if err != nil {
		return ErrInvalid{Field: "connections", Reason: err.Error()}
	}
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
		source := nodes[edge.From]
		maxPort := 0
		if source.Condition != nil {
			maxPort = 1
		}
		if source.FilterCondition != nil {
			maxPort = 1
		}
		if source.LoopBatchSize > 0 {
			maxPort = 1
		}
		for _, rule := range source.SwitchRules {
			if rule.OutputPort > maxPort {
				maxPort = rule.OutputPort
			}
		}
		if source.ErrorOutput {
			maxPort++
		}
		if edge.OutputPort > maxPort {
			return ErrInvalid{Field: "connections.main", Reason: fmt.Sprintf("output port %d is outside the modeled output set for node type %q", edge.OutputPort, source.Type)}
		}
		if !feedbackEdges[edge] {
			indegree[edge.To]++
			outgoing[edge.From] = append(outgoing[edge.From], edge.To)
		}
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
			if node.Condition != nil {
				entry["condition"] = node.Condition
			}
			if node.FilterCondition != nil {
				entry["filter_condition"] = node.FilterCondition
			}
			if len(node.SwitchRules) > 0 {
				entry["switch_rules"] = node.SwitchRules
			}
			if node.SetValues != nil {
				entry["set_values"] = node.SetValues
				entry["set_include_input"] = node.SetIncludeInput
			}
			if node.SetProjections != nil {
				entry["set_projections"] = node.SetProjections
				entry["set_include_input"] = node.SetIncludeInput
			}
			if node.SetExpressions != nil {
				entry["set_expressions"] = node.SetExpressions
				entry["set_include_input"] = node.SetIncludeInput
			}
			if node.SplitField != "" {
				entry["split_field"] = node.SplitField
			}
			if node.SafeCode != "" {
				digest := sha256.Sum256([]byte(node.SafeCode))
				entry["safe_code_sha256"] = hex.EncodeToString(digest[:])
			}
			if len(node.AggregateFields) > 0 {
				entry["aggregate_fields"] = node.AggregateFields
				entry["aggregate_keep_missing"] = node.AggregateKeepMissing
				entry["aggregate_merge_lists"] = node.AggregateMergeLists
				entry["aggregate_disable_dot_notation"] = node.AggregateDisableDotNotation
			}
			if node.WaitSeconds > 0 {
				entry["wait_seconds"] = node.WaitSeconds
			}
			if node.WaitAmountPath != "" {
				entry["wait_amount_path"] = node.WaitAmountPath
				entry["wait_unit"] = node.WaitUnit
			}
			if node.LoopBatchSize > 0 {
				entry["loop_batch_size"] = node.LoopBatchSize
			}
			if node.ErrorOutput {
				entry["error_output"] = true
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
