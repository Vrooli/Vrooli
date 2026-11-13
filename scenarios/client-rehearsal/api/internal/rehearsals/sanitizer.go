package rehearsals

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
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
		model.ExecuteOnce, _ = node["executeOnce"].(bool)
		model.AlwaysOutputData, _ = node["alwaysOutputData"].(bool)
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
		for connectionType, rawTargets := range entry {
			if connectionType == "main" {
				continue
			}
			if connectionType != "ai_embedding" && connectionType != "ai_textSplitter" && connectionType != "ai_document" {
				return Workflow{}, ErrInvalid{Field: "connections", Reason: "unsupported non-main connection type"}
			}
			outputs, _ := rawTargets.([]any)
			for outputPort, rawPortTargets := range outputs {
				targets, _ := rawPortTargets.([]any)
				for _, rawTarget := range targets {
					link, _ := rawTarget.(map[string]any)
					targetName, _ := link["node"].(string)
					to := byName[targetName]
					if to == "" {
						return Workflow{}, ErrInvalid{Field: "connections", Reason: "AI connection references a missing node"}
					}
					workflow.AIConnections = append(workflow.AIConnections, AIConnection{From: from, To: to, Type: connectionType, OutputPort: outputPort, InputPort: integer(link["index"])})
				}
			}
		}
	}
	if err := validateLangChainReferences(workflow); err != nil {
		return Workflow{}, err
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
		if typeName == "n8n-nodes-base.n8nTrigger" {
			// The pinned instance event trigger supports several vendor events. This
			// simulator models only the public `init` event (instance started), and
			// refuses other event shapes rather than flattening them into the same
			// trigger kind.
			events, _ := params["events"].([]any)
			if !onlyKeys(params, "events") || len(events) != 1 || events[0] != "init" {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.events", Reason: "n8n instance trigger supports only the bounded init event"}
			}
		}
		node.Trigger = kind
		return node, nil
	}
	switch typeName {
	case "@n8n/n8n-nodes-langchain.embeddingsOpenAi":
		if !onlyKeys(params, "options") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "OpenAI Embeddings accepts only bounded options"}
		}
		options, _ := params["options"].(map[string]any)
		if len(options) != 0 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "synthetic embeddings do not accept provider or request options"}
		}
		node.LangChain = &LangChainNode{Kind: "embeddings_openai"}
	case "@n8n/n8n-nodes-langchain.textSplitterRecursiveCharacterTextSplitter":
		if !onlyKeys(params, "options", "chunkSize", "chunkOverlap") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "recursive text splitter accepts only chunkSize, chunkOverlap, and empty options"}
		}
		options, _ := params["options"].(map[string]any)
		if len(options) != 0 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "synthetic recursive splitter accepts only default separators"}
		}
		chunkSize, sizeOK := positiveBoundedInteger(params["chunkSize"], 4000)
		if params["chunkSize"] == nil {
			chunkSize, sizeOK = 1000, true
		}
		chunkOverlap, overlapOK := nonNegativeBoundedInteger(params["chunkOverlap"], 1000)
		if params["chunkOverlap"] == nil {
			chunkOverlap, overlapOK = 200, true
		}
		if !sizeOK || !overlapOK || chunkOverlap >= chunkSize {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "chunk size must be 1..4000 and overlap must be smaller and at most 1000"}
		}
		node.LangChain = &LangChainNode{Kind: "recursive_text_splitter", ChunkSize: chunkSize, ChunkOverlap: chunkOverlap}
	case "@n8n/n8n-nodes-langchain.documentDefaultDataLoader":
		if !onlyKeys(params, "options", "jsonData", "jsonMode", "textSplittingMode") || params["jsonMode"] != "expressionData" || params["textSplittingMode"] != "custom" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Default Data Loader supports expressionData with custom text splitting only"}
		}
		dataRef, ok := parseNodeValueReference(params["jsonData"])
		if !ok || dataRef.NodeName == "" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.jsonData", Reason: "data loader requires a safe node-item JSON path reference"}
		}
		options, _ := params["options"].(map[string]any)
		if !onlyKeys(options, "metadata") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "data loader accepts only bounded metadata references"}
		}
		metadata := map[string]NodeValueRef{}
		if rawMetadata, exists := options["metadata"]; exists {
			metadataObject, ok := rawMetadata.(map[string]any)
			if !ok || !onlyKeys(metadataObject, "metadataValues") {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.metadata", Reason: "metadata must contain only metadataValues"}
			}
			values, _ := metadataObject["metadataValues"].([]any)
			if len(values) > 32 {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.metadata.metadataValues", Reason: "metadata values are limited to 32 fields"}
			}
			for _, rawValue := range values {
				entry, ok := rawValue.(map[string]any)
				name, nameOK := entry["name"].(string)
				ref, refOK := parseNodeValueReference(entry["value"])
				if !ok || !nameOK || !safeSyntheticFieldName(name) || !refOK || ref.NodeName == "" {
					return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.metadata.metadataValues", Reason: "metadata values must be safe names referencing one node item's JSON field"}
				}
				metadata[name] = ref
			}
		}
		node.LangChain = &LangChainNode{Kind: "default_data_loader", DataReference: &dataRef, MetadataRefs: metadata}
	case "@n8n/n8n-nodes-langchain.vectorStorePinecone":
		if !onlyKeys(params, "mode", "options", "pineconeIndex") || params["mode"] != "insert" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Pinecone simulator supports only insert with a declared index and namespace"}
		}
		indexObject, ok := params["pineconeIndex"].(map[string]any)
		if !ok || !onlyKeys(indexObject, "__rl", "mode", "value") || indexObject["mode"] != "id" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.pineconeIndex", Reason: "Pinecone index must use a bounded id selector"}
		}
		indexRef, ok := parseNodeValueReference(indexObject["value"])
		if !ok {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.pineconeIndex.value", Reason: "Pinecone index must be a static value or safe node-item reference"}
		}
		options, _ := params["options"].(map[string]any)
		if !onlyKeys(options, "pineconeNamespace") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "Pinecone accepts only a bounded namespace selector"}
		}
		namespaceRef, ok := parseNodeValueReference(options["pineconeNamespace"])
		if !ok {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.pineconeNamespace", Reason: "namespace must be a static value or safe node-item reference"}
		}
		node.LangChain = &LangChainNode{Kind: "pinecone_vector_store", Index: &indexRef, Namespace: &namespaceRef}
	case "n8n-nodes-base.httpRequest":
		request, expression, err := normalizeHTTPNode(id, name, params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.Request = &request
		node.RequestURLExpression = expression
	case "n8n-nodes-base.if":
		condition, err := normalizeCondition(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.Condition = &condition
	case "n8n-nodes-base.filter":
		conditions, combinator, err := normalizeFilter(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.FilterConditions = conditions
		node.FilterCombinator = combinator
		if len(conditions) == 1 {
			node.FilterCondition = &conditions[0]
		}
	case "n8n-nodes-base.switch":
		rules, err := normalizeSwitch(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.SwitchRules = rules
		if options, ok := params["options"].(map[string]any); ok {
			if !onlyKeys(options, "fallbackOutput", "renameFallbackOutput") {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "Switch synthetic mode accepts only a bounded fallback output selector"}
			}
			if fallback, ok := options["fallbackOutput"].(string); ok && fallback != "" && fallback != "none" {
				if fallback != "extra" {
					return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.fallbackOutput", Reason: "Switch fallback supports only the extra output"}
				}
				node.SwitchFallbackOutput = true
			}
			if label, ok := options["renameFallbackOutput"].(string); ok && (len(label) > 100 || strings.ContainsAny(label, "\x00\r\n")) {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options.renameFallbackOutput", Reason: "Switch fallback label is outside its bound"}
			}
		}
	case "n8n-nodes-base.noOp":
		// A no-op has deterministic pass-through semantics.
	case "n8n-nodes-base.set":
		mode, _ := params["mode"].(string)
		if mode == "" && (params["assignments"] != nil || params["fields"] != nil) {
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
			if err != nil && params["fields"] != nil {
				values, projections, expressions, includeInput, err = normalizeSetFields(params)
			}
			if err != nil && params["values"] != nil {
				values, projections, expressions, includeInput, err = normalizeSetValueGroups(params)
			}
			if err != nil {
				return WorkflowNode{}, err
			}
			node.SetValues = values
			node.SetProjections = projections
			node.SetExpressions = expressions
			node.SetIncludeInput = includeInput
		} else {
			if params["values"] != nil {
				values, projections, expressions, includeInput, err := normalizeSetValueGroups(params)
				if err != nil {
					return WorkflowNode{}, err
				}
				node.SetValues, node.SetProjections, node.SetExpressions, node.SetIncludeInput = values, projections, expressions, includeInput
			} else {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.mode", Reason: "Set accepts only bounded manual assignments or a raw JSON-path projection"}
			}
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
	case "n8n-nodes-base.code", "n8n-nodes-base.function":
		if typeName == "n8n-nodes-base.function" && typeVersion != 1 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.typeVersion", Reason: "legacy Function supports pinned version 1 only"}
		}
		if !onlyKeys(params, "jsCode", "functionCode", "mode", "language") {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Code accepts only source, mode, and language selectors"}
		}
		mode, _ := params["mode"].(string)
		if mode != "" && mode != "runOnceForAllItems" && mode != "runOnceForEachItem" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.mode", Reason: "safe Code supports only bounded runOnceForAllItems or runOnceForEachItem mode"}
		}
		node.SafeCodeRunOnceForEachItem = mode == "runOnceForEachItem"
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
		var parseErr error
		if typeName == "n8n-nodes-base.function" {
			_, parseErr = parseSafeLegacyFunction(source)
			node.SafeCodeLegacyFunction = true
		} else {
			_, parseErr = parseSafeCodeMode(source, false, node.SafeCodeRunOnceForEachItem)
		}
		if parseErr != nil {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.jsCode", Reason: "Code node is outside the bounded safe subset: " + parseErr.Error()}
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
		// RSS input is always resolved into the synthetic service adapter below.
		// The source URL is passed only as bounded inert input to the simulator;
		// no HTTP client ever opens it.
		node.Request = &Step{ID: id, Operation: typeName + ":feed.read", NodeType: typeName, Resource: "feed", Action: "read", Method: "GET", Host: "rss.synthetic.invalid", URL: "/v1/feed"}
		node.RequestReplacesInput = true
	case "n8n-nodes-base.aggregate":
		fields, keepMissing, mergeLists, disableDotNotation, err := normalizeAggregate(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.AggregateFields = fields
		node.AggregateKeepMissing = keepMissing
		node.AggregateMergeLists = mergeLists
		node.AggregateDisableDotNotation = disableDotNotation
	case "n8n-nodes-base.summarize":
		groupBy, countFields, err := normalizeSummarize(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.SummarizeGroupBy = groupBy
		node.SummarizeCountFields = countFields
	case "n8n-nodes-base.sort":
		fields, err := normalizeSort(params)
		if err != nil {
			return WorkflowNode{}, err
		}
		node.SortFields = fields
	case "n8n-nodes-base.merge":
		if !onlyKeys(params, "mode", "useDataOfInput") || params["mode"] != "chooseBranch" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Merge currently supports only a bounded choose-branch input selector"}
		}
		selected, ok := positiveBoundedInteger(params["useDataOfInput"], 2)
		if !ok {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.useDataOfInput", Reason: "Merge choose-branch must select input one or two"}
		}
		node.MergeUseDataOfInput = selected
	case "n8n-nodes-base.extractFromFile":
		if typeVersion != 1 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.typeVersion", Reason: "Extract From File supports pinned version 1 only"}
		}
		if !onlyKeys(params, "operation", "destinationKey", "binaryPropertyName", "options") || params["operation"] != "fromJson" {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Extract From File supports only bounded JSON decoding from one binary property"}
		}
		if options, ok := params["options"].(map[string]any); ok && len(options) != 0 {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.options", Reason: "Extract From File JSON options are unsupported"}
		}
		property, _ := params["binaryPropertyName"].(string)
		destination, _ := params["destinationKey"].(string)
		if property == "" || len(property) > 128 || strings.ContainsAny(property, "\x00\r\n") || !safeSyntheticFieldName(destination) {
			return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Extract From File requires bounded binary and destination property names"}
		}
		node.ExtractFromFilePropertyName = property
		node.ExtractFromFileDestination = destination
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
		if typeName == "n8n-nodes-base.googleSheets" && resource == "sheet" && action == "" && typeVersion >= 4 && typeVersion <= 4.7 {
			// Pinned n8n 2.48.0 GoogleSheetsV2 uses Sheet.resource's default
			// operation "read" across its supported v4 node versions.
			// Source: https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Google/Sheet/v2/actions/sheet/Sheet.resource.ts
			action = "read"
		}
		if typeName == "n8n-nodes-base.dataTable" && typeVersion == 1.1 && resource == "" {
			// Pinned n8n 2.48.0 Data Table defaults resource to row.
			// Source: https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/DataTable/DataTable.node.ts
			resource = "row"
		}
		if typeName == "n8n-nodes-base.gmail" {
			if resource == "" {
				resource = "message"
			}
			if action == "" {
				action = "send"
			}
		}
		if typeName == "n8n-nodes-base.n8n" && typeVersion == 1 {
			// Pinned n8n node defaults Resource to Workflow and its Operation to Get Many.
			// Source: https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/N8n/N8n.node.ts
			// and WorkflowDescription.ts. Only an empty bounded filter is admitted here.
			if resource == "" {
				resource = "workflow"
			}
			if action == "" {
				action = "getAll"
			}
			if filters, ok := params["filters"].(map[string]any); ok && len(filters) != 0 {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters.filters", Reason: "synthetic n8n workflow listing supports only empty filters"}
			}
		}
		if typeName == "n8n-nodes-base.discord" && typeVersion == 2 && params["authentication"] == "webhook" && resource == "" && action == "" {
			content, _ := params["content"].(string)
			options, optionsOK := params["options"].(map[string]any)
			if content == "" || len(content) > 2000 || strings.ContainsRune(content, '\x00') || !optionsOK || len(options) != 0 {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Discord webhook simulation requires bounded message content and empty default options"}
			}
			// The v2 webhook UI hides its resource selector. Keep a normalized
			// semantic resource for profile matching; the selected operation is the
			// source-declared sendLegacy default.
			// Sources: https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Discord/v2/actions/versionDescription.ts
			// and https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Discord/v2/actions/webhook/index.ts
			resource, action = "webhook", "sendLegacy"
		}
		if typeName == "n8n-nodes-base.telegram" && typeVersion == 1.2 && resource == "" && action == "" {
			text, _ := params["text"].(string)
			chatID, _ := params["chatId"].(string)
			additional, additionalOK := params["additionalFields"].(map[string]any)
			if text == "" || len(text) > 4096 || strings.ContainsRune(text, '\x00') || chatID == "" || len(chatID) > 128 || strings.ContainsAny(chatID, "\x00\r\n") || !additionalOK || len(additional) != 0 {
				return WorkflowNode{}, ErrInvalid{Field: "nodes.parameters", Reason: "Telegram simulation requires bounded message text, a static chat id, and empty default fields"}
			}
			// Pinned n8n 2.48.0 Telegram.node.ts defaults resource to message and
			// message operation to sendMessage for v1.2.
			// https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Telegram/Telegram.node.ts
			resource, action = "message", "sendMessage"
		}
		if typeName == "n8n-nodes-base.gitlab" && typeVersion == 1 && resource == "file" && action == "" {
			// GitLab 2.48.0's File resource defaults operation to create. Resolve this
			// exported UI default only; all effects still require a synthetic profile.
			// https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Gitlab/Gitlab.node.ts
			action = "create"
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
		if typeName == "n8n-nodes-base.n8n" && resource == "workflow" && action == "getAll" {
			// The n8n Workflow Get Many operation returns the listed workflow items;
			// it does not merge the caller's input fields into each workflow.
			node.RequestReplacesInput = true
		}
		// Preserve a small, safe set of static service selectors so a published
		// API description can distinguish the target table/database without
		// retaining caller data or workflow-specific routing logic.
		if tableID, ok := params["tableId"].(string); ok && tableID != "" && len(tableID) <= 128 && !strings.Contains(tableID, "={{") && !strings.ContainsAny(tableID, "\x00\r\n") {
			request.Input = map[string]any{"tableId": tableID}
		}
		if typeName == "n8n-nodes-base.googleSheets" && resource == "sheet" && action == "read" {
			if locator, ok := params["sheetName"].(map[string]any); ok {
				if value, ok := locator["value"].(string); ok && value != "" && len(value) <= 128 && !strings.ContainsAny(value, "\x00\r\n") && !strings.Contains(value, "={{") {
					if request.Input == nil {
						request.Input = map[string]any{}
					}
					request.Input["sheetName"] = value
				}
			}
			node.RequestReplacesInput = true
		}
		node.Request = request
	}
	return node, nil
}

func positiveBoundedInteger(value any, max int) (int, bool) {
	number, ok := value.(float64)
	if !ok || number < 1 || number > float64(max) || number != float64(int(number)) {
		return 0, false
	}
	return int(number), true
}

func nonNegativeBoundedInteger(value any, max int) (int, bool) {
	number, ok := value.(float64)
	if !ok || number < 0 || number > float64(max) || number != float64(int(number)) {
		return 0, false
	}
	return int(number), true
}

// parseNodeValueReference retains only literal data or a simple n8n item path.
// It deliberately does not preserve or evaluate general expressions.
func parseNodeValueReference(raw any) (NodeValueRef, bool) {
	if literal, ok := raw.(string); ok {
		value := strings.TrimSpace(literal)
		if strings.HasPrefix(value, "={{") && strings.HasSuffix(value, "}}") {
			value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "={{"), "}}"))
			value = strings.TrimPrefix(value, "String(")
			value = strings.TrimSuffix(value, ")")
			if !strings.HasPrefix(value, "$(") {
				return NodeValueRef{}, false
			}
			end := strings.Index(value, ")")
			if end < 3 {
				return NodeValueRef{}, false
			}
			nodeName := strings.Trim(value[2:end], "'\"")
			tail := value[end+1:]
			for _, selector := range []string{".item.json.", ".first().json."} {
				if strings.HasPrefix(tail, selector) {
					path := strings.TrimPrefix(tail, selector)
					if nodeName != "" && validJSONPath(path) {
						return NodeValueRef{NodeName: nodeName, Path: path}, true
					}
				}
			}
			return NodeValueRef{}, false
		}
		if len(value) > 256 || strings.ContainsAny(value, "\x00\r\n") || strings.Contains(value, "={{") {
			return NodeValueRef{}, false
		}
		return NodeValueRef{Literal: value}, true
	}
	if number, ok := raw.(float64); ok && number >= 0 && number < 1e12 {
		return NodeValueRef{Literal: number}, true
	}
	return NodeValueRef{}, false
}

func validateLangChainReferences(workflow Workflow) error {
	names := make(map[string]bool, len(workflow.Nodes))
	for _, node := range workflow.Nodes {
		names[node.Name] = true
	}
	for _, node := range workflow.Nodes {
		if node.LangChain == nil {
			continue
		}
		refs := []*NodeValueRef{node.LangChain.DataReference, node.LangChain.Index, node.LangChain.Namespace}
		for _, ref := range refs {
			if ref != nil && ref.NodeName != "" && !names[ref.NodeName] {
				return ErrInvalid{Field: "nodes.parameters", Reason: "LangChain value reference names a missing workflow node"}
			}
		}
		for _, ref := range node.LangChain.MetadataRefs {
			if ref.NodeName != "" && !names[ref.NodeName] {
				return ErrInvalid{Field: "nodes.parameters.options.metadata", Reason: "LangChain metadata reference names a missing workflow node"}
			}
		}
	}
	return nil
}

func normalizeSetFields(params map[string]any) (map[string]any, map[string]string, map[string]string, bool, error) {
	if !onlyKeys(params, "fields", "include", "options") {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters", Reason: "legacy Set fields accepts only fields, include and empty options"}
	}
	fields, ok := params["fields"].(map[string]any)
	if !ok || !onlyKeys(fields, "values") {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.fields", Reason: "legacy Set fields must contain only a bounded values list"}
	}
	items, _ := fields["values"].([]any)
	if len(items) == 0 || len(items) > 64 {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.fields.values", Reason: "legacy Set requires 1 to 64 bounded values"}
	}
	assignments := make([]any, 0, len(items))
	for _, raw := range items {
		field, ok := raw.(map[string]any)
		if !ok || !onlyKeys(field, "name", "type", "stringValue", "numberValue", "booleanValue", "objectValue") {
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.fields.values", Reason: "each legacy Set value must have a name and one supported value field"}
		}
		name, _ := field["name"].(string)
		present := 0
		var kind string
		var value any
		declaredType, _ := field["type"].(string)
		for key, candidate := range field {
			switch key {
			case "type":
				continue
			case "stringValue":
				kind, value = "string", candidate
				present++
			case "numberValue":
				kind, value = "number", candidate
				present++
			case "booleanValue":
				kind, value = "boolean", candidate
				present++
			case "objectValue":
				kind, value = "object", candidate
				present++
			}
		}
		if name == "" || present != 1 {
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.fields.values", Reason: "legacy Set values require a name and exactly one supported value"}
		}
		if declaredType != "" {
			expected := kind + "Value"
			if kind == "expression" {
				expected = "objectValue"
			}
			if declaredType != expected {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.fields.values.type", Reason: "declared Set type must match its single value field"}
			}
		}
		if text, ok := value.(string); ok && strings.HasPrefix(text, "=") {
			expression, err := normalizeLegacySetExpression(text)
			if err != nil {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.fields.values", Reason: "legacy Set expression is outside the bounded subset: " + err.Error()}
			}
			value, kind = expression, "expression"
		}
		assignments = append(assignments, map[string]any{"name": name, "type": kind, "value": value})
	}
	include := false
	switch params["include"] {
	case nil, "none":
	case "all":
		include = true
	default:
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.include", Reason: "legacy Set include supports only none or all"}
	}
	if opts, ok := params["options"].(map[string]any); !ok || len(opts) != 0 {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.options", Reason: "legacy Set options must be empty"}
	}
	return normalizeSetAssignments(map[string]any{"assignments": map[string]any{"assignments": assignments}, "includeOtherFields": include})
}

// normalizeSetValueGroups handles the n8n Set node's older grouped-value
// representation. Each type bucket is only a transport shape: expressions are
// still validated and evaluated by the same bounded assignment interpreter.
func normalizeSetValueGroups(params map[string]any) (map[string]any, map[string]string, map[string]string, bool, error) {
	if !onlyKeys(params, "values", "options", "keepOnlySet") {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters", Reason: "Set value groups accept only values, empty options and keepOnlySet"}
	}
	groups, ok := params["values"].(map[string]any)
	if !ok || !onlyKeys(groups, "string", "number", "boolean", "array", "object") {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.values", Reason: "Set value groups require known bounded type buckets"}
	}
	assignments := make([]any, 0, 16)
	for _, kind := range []string{"string", "number", "boolean", "array", "object"} {
		entries, exists := groups[kind]
		if !exists {
			continue
		}
		items, ok := entries.([]any)
		if !ok || len(items) > 64 || len(assignments)+len(items) > 64 {
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.values", Reason: "Set type buckets must be arrays with at most 64 assignments"}
		}
		for _, raw := range items {
			entry, ok := raw.(map[string]any)
			if !ok || !onlyKeys(entry, "name", "value") {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.values", Reason: "each Set type-bucket entry requires only name and value"}
			}
			name, _ := entry["name"].(string)
			value, hasValue := entry["value"]
			if !hasValue {
				switch kind {
				case "string":
					value = ""
				case "number":
					value = float64(0)
				case "boolean":
					value = false
				case "array":
					value = "[]"
				case "object":
					value = "{}"
				}
			}
			assignments = append(assignments, map[string]any{"name": name, "type": kind, "value": value})
		}
	}
	if len(assignments) == 0 {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.values", Reason: "Set requires at least one value assignment"}
	}
	include := false
	if keep, ok := params["keepOnlySet"].(bool); ok {
		include = !keep
	}
	if options, ok := params["options"].(map[string]any); ok && len(options) != 0 {
		return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.options", Reason: "Set value-group options must be empty"}
	}
	return normalizeSetAssignments(map[string]any{"assignments": map[string]any{"assignments": assignments}, "includeOtherFields": include})
}

func normalizeLegacySetExpression(source string) (string, error) {
	if strings.HasPrefix(source, "={{") && strings.HasSuffix(source, "}}") {
		return normalizeSafeExpression(source)
	}
	if !strings.HasPrefix(source, "=") || !strings.Contains(source, "{{") || strings.Contains(source, "}}") == false {
		return "", fmt.Errorf("mixed template must use bounded {{ expression }} segments")
	}
	body := source[1:]
	var expression strings.Builder
	for len(body) > 0 {
		start := strings.Index(body, "{{")
		if start < 0 {
			if strings.Contains(body, "}}") {
				return "", fmt.Errorf("unmatched template delimiter")
			}
			if expression.Len() > 0 {
				expression.WriteString(" + ")
			}
			expression.WriteString(strconv.Quote(body))
			body = ""
			break
		}
		if strings.Contains(body[:start], "}}") {
			return "", fmt.Errorf("unmatched template delimiter")
		}
		if start > 0 {
			if expression.Len() > 0 {
				expression.WriteString(" + ")
			}
			expression.WriteString(strconv.Quote(body[:start]))
		}
		end := strings.Index(body[start+2:], "}}")
		if end < 0 {
			return "", fmt.Errorf("unclosed template expression")
		}
		part := strings.TrimSpace(body[start+2 : start+2+end])
		if part == "" {
			return "", fmt.Errorf("empty template expression")
		}
		if expression.Len() > 0 {
			expression.WriteString(" + ")
		}
		expression.WriteString("String(")
		expression.WriteString(part)
		expression.WriteByte(')')
		body = body[start+2+end+2:]
	}
	if expression.Len() == 0 {
		return "", fmt.Errorf("template expression is empty")
	}
	expr, err := normalizeSafeExpression(expression.String())
	if err != nil {
		return "", err
	}
	return expr, nil
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
		if exprText, ok := value.(string); ok && (kind == "expression" || strings.Contains(exprText, "={{")) {
			expression := exprText
			var err error
			if kind != "expression" {
				expression, err = normalizeSafeExpression(exprText)
			}
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
		case "array":
			text, ok := value.(string)
			if !ok || len(text) > 4096 {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set array values must be bounded JSON array literals"}
			}
			var items []any
			if err := json.Unmarshal([]byte(text), &items); err != nil || len(items) > 64 {
				return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set array values must be JSON arrays with at most 64 primitive items"}
			}
			for _, item := range items {
				switch item.(type) {
				case nil, bool, float64, string:
				default:
					return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set array items must be JSON primitives"}
				}
				if stringItem, ok := item.(string); ok && (len(stringItem) > 512 || strings.ContainsRune(stringItem, '\x00')) {
					return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set array strings must be bounded"}
				}
			}
			values[name] = items
		case "object":
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set object values must use a safe expression"}
		default:
			return nil, nil, nil, false, ErrInvalid{Field: "nodes.parameters.assignments", Reason: "Set supports bounded string, number, boolean and primitive array values"}
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

func normalizeSummarize(params map[string]any) ([]string, []string, error) {
	if !onlyKeys(params, "fieldsToSplitBy", "fieldsToSummarize", "options") {
		return nil, nil, ErrInvalid{Field: "nodes.parameters", Reason: "Summarize supports bounded group-by fields, field counts and empty options"}
	}
	if options, ok := params["options"].(map[string]any); !ok || len(options) != 0 {
		return nil, nil, ErrInvalid{Field: "nodes.parameters.options", Reason: "Summarize options must be empty"}
	}
	groupText, _ := params["fieldsToSplitBy"].(string)
	groupBy := []string{}
	for _, field := range strings.Split(groupText, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if !validJSONPath(field) || len(groupBy) >= 16 {
			return nil, nil, ErrInvalid{Field: "nodes.parameters.fieldsToSplitBy", Reason: "Summarize group-by fields must be bounded JSON paths"}
		}
		groupBy = append(groupBy, field)
	}
	container, ok := params["fieldsToSummarize"].(map[string]any)
	if !ok || !onlyKeys(container, "values") {
		return nil, nil, ErrInvalid{Field: "nodes.parameters.fieldsToSummarize", Reason: "Summarize requires bounded count fields"}
	}
	values, ok := container["values"].([]any)
	if !ok || len(values) == 0 || len(values) > 32 {
		return nil, nil, ErrInvalid{Field: "nodes.parameters.fieldsToSummarize.values", Reason: "Summarize requires one to 32 count fields"}
	}
	countFields := make([]string, 0, len(values))
	for _, raw := range values {
		field, ok := raw.(map[string]any)
		name, _ := field["field"].(string)
		if !ok || !onlyKeys(field, "field") || !validJSONPath(name) {
			return nil, nil, ErrInvalid{Field: "nodes.parameters.fieldsToSummarize.values", Reason: "Summarize supports only bounded count descriptors"}
		}
		countFields = append(countFields, name)
	}
	return groupBy, countFields, nil
}

func normalizeSort(params map[string]any) ([]SortField, error) {
	if !onlyKeys(params, "sortFieldsUi", "options") {
		return nil, ErrInvalid{Field: "nodes.parameters", Reason: "Sort accepts only bounded sort fields and empty options"}
	}
	if options, ok := params["options"].(map[string]any); !ok || len(options) != 0 {
		return nil, ErrInvalid{Field: "nodes.parameters.options", Reason: "Sort options must be empty"}
	}
	container, ok := params["sortFieldsUi"].(map[string]any)
	if !ok || !onlyKeys(container, "sortField") {
		return nil, ErrInvalid{Field: "nodes.parameters.sortFieldsUi", Reason: "Sort requires bounded field descriptors"}
	}
	values, ok := container["sortField"].([]any)
	if !ok || len(values) == 0 || len(values) > 16 {
		return nil, ErrInvalid{Field: "nodes.parameters.sortFieldsUi.sortField", Reason: "Sort supports one to 16 fields"}
	}
	fields := make([]SortField, 0, len(values))
	for _, raw := range values {
		entry, ok := raw.(map[string]any)
		name, _ := entry["fieldName"].(string)
		order, _ := entry["order"].(string)
		if !ok || !onlyKeys(entry, "fieldName", "order") || !validJSONPath(name) || order != "ascending" && order != "descending" {
			return nil, ErrInvalid{Field: "nodes.parameters.sortFieldsUi.sortField", Reason: "Sort fields require a bounded JSON path and ascending or descending order"}
		}
		fields = append(fields, SortField{Path: name, Descending: order == "descending"})
	}
	return fields, nil
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
			if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') && value[len(value)-1] == value[0] {
				key := value[1 : len(value)-1]
				if !validQuotedJSONKey(key) {
					return nil, false
				}
				result = append(result, jsonPathSegment{key: key})
				pos = end + 1
				if len(result) > 8 {
					return nil, false
				}
				continue
			}
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
	return (strings.HasPrefix(typeName, "n8n-nodes-base.") && !strings.HasSuffix(typeName, "Trigger")) || typeName == "@n8n/n8n-nodes-langchain.openAi" || typeName == "@n8n/n8n-nodes-langchain.vectorStorePinecone"
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
	case "n8n-nodes-base.n8nTrigger":
		return TriggerEvent, true
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

func normalizeHTTPNode(id, name string, params map[string]any) (Step, string, error) {
	rawURL, _ := params["url"].(string)
	if strings.Contains(rawURL, "={{") {
		if _, err := parseSafeSyntheticURLExpression(rawURL); err != nil {
			return Step{}, "", ErrInvalid{Field: "nodes.parameters.url", Reason: "HTTP Request URL expression is outside the bounded synthetic URL subset"}
		}
		method, _ := params["method"].(string)
		if method == "" {
			method = "GET"
		}
		method = strings.ToUpper(method)
		if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			return Step{}, "", ErrInvalid{Field: "nodes.parameters.method", Reason: "unsupported HTTP method"}
		}
		return Step{ID: id, Operation: method + " dynamic synthetic URL", Method: method}, rawURL, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return Step{}, "", ErrInvalid{Field: "nodes.parameters.url", Reason: "HTTP Request nodes require an absolute HTTP(S) URL or a bounded synthetic URL expression"}
	}
	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return Step{}, "", ErrInvalid{Field: "nodes.parameters.method", Reason: "unsupported HTTP method"}
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	return Step{ID: id, Operation: method + " " + path, Method: method, Host: parsed.Hostname(), URL: path}, "", nil
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

func normalizeFilter(params map[string]any) ([]Condition, string, error) {
	if !onlyKeys(params, "conditions", "options") {
		return nil, "", ErrInvalid{Field: "nodes.parameters", Reason: "Filter accepts only conditions and options"}
	}
	if options, ok := params["options"].(map[string]any); !ok || len(options) != 0 {
		return nil, "", ErrInvalid{Field: "nodes.parameters.options", Reason: "Filter options must be empty"}
	}
	root, ok := params["conditions"].(map[string]any)
	if !ok || !onlyKeys(root, "combinator", "conditions", "options") {
		return nil, "", ErrInvalid{Field: "nodes.parameters.conditions", Reason: "Filter requires a bounded condition object"}
	}
	combinator, _ := root["combinator"].(string)
	if combinator != "and" && combinator != "or" {
		return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.combinator", Reason: "Filter combinator must be and or or"}
	}
	rootOptions, ok := root["options"].(map[string]any)
	if !ok || !onlyKeys(rootOptions, "version", "caseSensitive", "leftValue", "typeValidation") {
		return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.options", Reason: "Filter condition options are unsupported"}
	}
	if rootOptions["caseSensitive"] != true || rootOptions["leftValue"] != "" || (rootOptions["typeValidation"] != "strict" && rootOptions["typeValidation"] != "loose") || (rootOptions["version"] != nil && rootOptions["version"] != float64(3)) {
		return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.options", Reason: "Filter options must use the supported case and type settings"}
	}
	conditions, ok := root["conditions"].([]any)
	if !ok || len(conditions) == 0 || len(conditions) > 8 {
		return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions", Reason: "Filter supports one to eight bounded conditions"}
	}
	result := make([]Condition, 0, len(conditions))
	for _, raw := range conditions {
		entry, ok := raw.(map[string]any)
		if !ok || !onlyKeys(entry, "id", "leftValue", "operator", "rightValue") {
			return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions", Reason: "Filter condition fields are unsupported"}
		}
		operator, ok := entry["operator"].(map[string]any)
		if !ok || !onlyKeys(operator, "type", "operation", "singleValue") {
			return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter condition operator is unsupported"}
		}
		typeName, _ := operator["type"].(string)
		if typeName == "boolean" && operator["singleValue"] != nil && operator["singleValue"] != true || typeName != "boolean" && typeName != "dateTime" {
			return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter supports bounded boolean and date-time comparisons"}
		}
		condition, err := conditionFromEntry(entry)
		if err != nil {
			return nil, "", err
		}
		if typeName == "boolean" {
			if condition.Operator == "equals" {
				if entry["rightValue"] != true && entry["rightValue"] != false {
					return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter boolean equality requires a boolean right value"}
				}
			} else if (condition.Operator != "true" && condition.Operator != "false") || entry["rightValue"] != (condition.Operator == "true") {
				return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter boolean predicate must be a matching true or false test"}
			}
		} else if condition.Operator != "dateAfterOrEquals" {
			return nil, "", ErrInvalid{Field: "nodes.parameters.conditions.conditions.operator", Reason: "Filter date-time supports after-or-equal comparisons"}
		}
		result = append(result, condition)
	}
	return result, combinator, nil
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
	if match := dateAfterNowExpression.FindStringSubmatch(strings.TrimSpace(left)); len(match) == 2 {
		path, nodeName, ok := parseJSONExpressionSource("={{" + strings.TrimSpace(match[1]) + "}}")
		if !ok || nodeName != "" {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.leftValue", Reason: "date comparison requires a bounded input JSON path"}
		}
		return Condition{Path: path, Operator: "dateAfterNow"}, nil
	}
	if match := numericRangeExpression.FindStringSubmatch(strings.TrimSpace(left)); len(match) == 5 {
		path, nodeName, ok := parseJSONExpressionSource("={{" + match[1] + "}}")
		secondPath, secondNode, secondOK := parseJSONExpressionSource("={{" + match[3] + "}}")
		lower, lowErr := strconv.ParseFloat(match[2], 64)
		upper, highErr := strconv.ParseFloat(match[4], 64)
		if !ok || !secondOK || nodeName != "" || secondNode != "" || secondPath != path || lowErr != nil || highErr != nil || lower >= upper {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.leftValue", Reason: "numeric range requires one JSON path and increasing numeric bounds"}
		}
		return Condition{Path: path, Operator: "numericRange", Value: []any{lower, upper}}, nil
	}
	operatorRoot, _ := entry["operator"].(map[string]any)
	operator, _ := operatorRoot["operation"].(string)
	right, _ := entry["rightValue"].(string)
	if operator == "equals" && strings.TrimSpace(right) == "={{ $now.format('yyyy-MM-dd') }}" {
		path, nodeName, ok := parseJSONExpressionSource(left)
		if !ok {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "date equality left value must use a simple JSON path expression"}
		}
		return Condition{Path: path, NodeName: nodeName, Operator: operator, NowFormat: "2006-01-02"}, nil
	}
	if strings.TrimSpace(left) == "={{ $input.all().length }}" {
		path := "$input.length"
		value, ok := entry["rightValue"]
		if !ok {
			value = entry["value2"]
		}
		numeric, valid := value.(float64)
		if !valid || operator != "gt" || numeric != 0 {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "input item count currently supports only greater-than-zero"}
		}
		return Condition{Path: path, Operator: operator, Value: value}, nil
	}
	path, nodeName, ok := parseJSONExpressionSource(left)
	if !ok {
		return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "condition must use a simple JSON path expression"}
	}
	if operatorRoot["type"] == "dateTime" && operator == "afterOrEquals" {
		leftPath, leftNode, ok := parseJSONExpressionSource(left)
		if !ok {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions", Reason: "date-time left value must use a simple JSON path"}
		}
		right, _ := entry["rightValue"].(string)
		match := dateTimeMinusExpression.FindStringSubmatch(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(right), "={{"), "}}")))
		if len(match) != 5 || !validJSONPath(match[2]) || !validJSONPath(match[4]) {
			return Condition{}, ErrInvalid{Field: "nodes.parameters.conditions.rightValue", Reason: "date-time cutoff must be a bounded node timestamp minus a node duration in minutes"}
		}
		return Condition{Path: leftPath, NodeName: leftNode, Operator: "dateAfterOrEquals", RightNodeName: match[1], RightPath: match[2], SubtractNodeName: match[3], SubtractPath: match[4], SubtractUnit: "minutes"}, nil
	}
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
	return Condition{Path: path, NodeName: nodeName, Operator: operator, Value: value}, nil
}

var dateTimeMinusExpression = regexp.MustCompile(`^\$\('([^']+)'\)\.first\(\)\.json\.([A-Za-z0-9_.-]+)\.toDateTime\(\)\.minus\(\$\('([^']+)'\)\.first\(\)\.json\.([A-Za-z0-9_.-]+), 'minutes'\)$`)
var dateAfterNowExpression = regexp.MustCompile(`^=\{\{\s*new Date\((\$json(?:\.[A-Za-z0-9_-]+|\[(?:"[^"]+"|'[^']+')\])+)\)\s*>\s*new Date\(\)\s*\}\}$`)
var numericRangeExpression = regexp.MustCompile(`^=\{\{\s*(\$json(?:\.[A-Za-z0-9_-]+|\[(?:"[^"]+"|'[^']+')\])*)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*&&\s*(\$json(?:\.[A-Za-z0-9_-]+|\[(?:"[^"]+"|'[^']+')\])*)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\}\}$`)

func normalizeSwitch(params map[string]any) ([]SwitchRule, error) {
	rulesRoot, _ := params["rules"].(map[string]any)
	values, _ := rulesRoot["values"].([]any)
	if len(values) == 0 {
		values, _ = rulesRoot["rules"].([]any)
	}
	if len(values) == 0 || len(values) > 32 {
		return nil, ErrInvalid{Field: "nodes.parameters.rules", Reason: "Switch requires 1 to 32 rules"}
	}
	legacyPath, legacyNode := "", ""
	if raw, _ := params["value1"].(string); raw != "" {
		var ok bool
		legacyPath, legacyNode, ok = parseJSONExpressionSource(raw)
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
		condition, err := switchCondition(rule, legacyPath, legacyNode)
		if err != nil {
			return nil, err
		}
		result = append(result, SwitchRule{Condition: condition, OutputPort: outputPort})
	}
	return result, nil
}

func switchCondition(rule map[string]any, legacyPath, legacyNode string) (Condition, error) {
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
	return Condition{Path: legacyPath, NodeName: legacyNode, Operator: "equals", Value: rule["value2"]}, nil
}

func parseJSONExpression(raw string) (string, bool) {
	path, _, ok := parseJSONExpressionSource(raw)
	return path, ok
}

func parseJSONExpressionSource(raw string) (string, string, bool) {
	expression := strings.TrimSpace(raw)
	if strings.HasPrefix(expression, "=") {
		expression = strings.TrimSpace(strings.TrimPrefix(expression, "="))
	}
	if !strings.HasPrefix(expression, "{{") || !strings.HasSuffix(expression, "}}") {
		return "", "", false
	}
	expression = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "{{"), "}}"))
	path, nodeName := "", ""
	if strings.HasPrefix(expression, "$json.") {
		path = "." + strings.TrimPrefix(expression, "$json.")
	} else if strings.HasPrefix(expression, "$json[") {
		path = strings.TrimPrefix(expression, "$json")
	} else if strings.HasPrefix(expression, "$node[") {
		jsonIndex := strings.Index(expression, "].json")
		if jsonIndex < 0 {
			return "", "", false
		}
		rawName := strings.TrimSpace(expression[len("$node["):jsonIndex])
		if len(rawName) < 2 || (rawName[0] != '\'' && rawName[0] != '"') || rawName[len(rawName)-1] != rawName[0] {
			return "", "", false
		}
		nodeName = rawName[1 : len(rawName)-1]
		if !safeCodeNodeName(nodeName) {
			return "", "", false
		}
		path = expression[jsonIndex+len("].json"):]
		if strings.HasPrefix(path, ".") {
			path = strings.TrimPrefix(path, ".")
		}
	} else {
		return "", "", false
	}
	parts := []string{}
	for len(path) > 0 {
		if strings.HasPrefix(path, `["`) || strings.HasPrefix(path, `['`) {
			quote := path[1]
			end := strings.Index(path[2:], string([]byte{quote, ']'}))
			if end < 0 {
				return "", "", false
			}
			part := path[2 : 2+end]
			if !validQuotedJSONKey(part) {
				return "", "", false
			}
			parts = append(parts, part)
			path = path[2+end+2:]
			continue
		}
		if strings.HasPrefix(path, "[") {
			end := strings.IndexByte(path, ']')
			if end < 0 || len(parts) == 0 {
				return "", "", false
			}
			rawIndex := path[1:end]
			index, err := strconv.Atoi(rawIndex)
			if err != nil || index < 0 || index > 100 {
				return "", "", false
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
				return "", "", false
			}
			parts = append(parts, part)
			path = path[len(part):]
			continue
		}
		if strings.HasPrefix(path, "[\"") {
			end := strings.Index(path[2:], "\"]")
			if end < 0 {
				return "", "", false
			}
			part := path[2 : 2+end]
			if !validPathPart(part) {
				return "", "", false
			}
			parts = append(parts, part)
			path = path[2+end+2:]
			continue
		}
		return "", "", false
	}
	if len(parts) == 0 {
		return "", "", false
	}
	pathParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if validPathPart(part) {
			pathParts = append(pathParts, part)
		} else {
			pathParts = append(pathParts, `['`+part+`']`)
		}
	}
	return strings.Join(pathParts, "."), nodeName, true
}

func validQuotedJSONKey(key string) bool {
	return key != "" && len(key) <= 64 && !strings.ContainsAny(key, "\x00\r\n[]'\"")
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
		if len(source.FilterConditions) > 0 {
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
		if source.SwitchFallbackOutput && len(source.SwitchRules) > maxPort {
			maxPort = len(source.SwitchRules)
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
	for _, edge := range workflow.AIConnections {
		source, target := nodes[edge.From], nodes[edge.To]
		if source.ID == "" || target.ID == "" || source.LangChain == nil || target.LangChain == nil {
			return ErrInvalid{Field: "connections", Reason: "AI connections must join supported LangChain nodes"}
		}
		valid := (edge.Type == "ai_embedding" && source.LangChain.Kind == "embeddings_openai" && target.LangChain.Kind == "pinecone_vector_store") ||
			(edge.Type == "ai_textSplitter" && source.LangChain.Kind == "recursive_text_splitter" && target.LangChain.Kind == "default_data_loader") ||
			(edge.Type == "ai_document" && source.LangChain.Kind == "default_data_loader" && target.LangChain.Kind == "pinecone_vector_store")
		if !valid || edge.OutputPort != 0 || edge.InputPort != 0 {
			return ErrInvalid{Field: "connections", Reason: "AI connection type or port is incompatible with its LangChain node pair"}
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
	for changed := true; changed; {
		changed = false
		for _, edge := range workflow.AIConnections {
			if reachable[edge.To] && !reachable[edge.From] {
				reachable[edge.From] = true
				changed = true
			}
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
				if len(node.Request.Input) > 0 {
					// Native operation selectors such as a Sheets tab change the
					// simulated effect even when method/path/type stay identical.
					entry["request_input"] = node.Request.Input
				}
			}
			if node.LangChain != nil {
				entry["langchain"] = node.LangChain
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
			if len(node.FilterConditions) > 0 {
				entry["filter_conditions"] = node.FilterConditions
				entry["filter_combinator"] = node.FilterCombinator
			}
			if len(node.SwitchRules) > 0 {
				entry["switch_rules"] = node.SwitchRules
				if node.SwitchFallbackOutput {
					entry["switch_fallback"] = true
				}
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
			if len(node.SummarizeCountFields) > 0 {
				entry["summarize_count_fields"] = node.SummarizeCountFields
				if len(node.SummarizeGroupBy) > 0 {
					entry["summarize_group_by"] = node.SummarizeGroupBy
				}
			}
			if len(node.SortFields) > 0 {
				entry["sort_fields"] = node.SortFields
			}
			if node.MergeUseDataOfInput > 0 {
				entry["merge_use_data_of_input"] = node.MergeUseDataOfInput
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
		aiEdges := append([]AIConnection(nil), workflow.AIConnections...)
		sort.Slice(aiEdges, func(i, j int) bool {
			if aiEdges[i].From == aiEdges[j].From {
				if aiEdges[i].Type == aiEdges[j].Type {
					return aiEdges[i].To < aiEdges[j].To
				}
				return aiEdges[i].Type < aiEdges[j].Type
			}
			return aiEdges[i].From < aiEdges[j].From
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
		return map[string]any{"nodes": nodes, "edges": edges, "ai_edges": aiEdges, "triggers": workflow.TriggerKinds, "error": workflow.ErrorWorkflowID, "children": children}
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

func safeSyntheticFieldName(name string) bool {
	if name == "" || len(name) > 128 || strings.ContainsAny(name, "\x00\r\n") {
		return false
	}
	switch name {
	case "__proto__", "prototype", "constructor", "caller", "callee", "arguments":
		return false
	}
	return true
}
