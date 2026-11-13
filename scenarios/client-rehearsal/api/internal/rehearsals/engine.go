package rehearsals

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxSubworkflowDepth = 16

// SyntheticEngine evaluates a validated acyclic workflow graph. All HTTP
// operations go through the injected simulator; the interpreter has no HTTP
// client, runtime endpoint, or credential surface.
type SyntheticEngine struct{}

func (engine SyntheticEngine) Execute(ctx context.Context, workflow Workflow, triggerNodeID string, simulator Simulator, injectFault bool, now time.Time) ([]TraceEvent, error) {
	execution, err := engine.ExecuteDetailed(ctx, workflow, triggerNodeID, simulator, injectFault, now)
	return execution.Trace, err
}

func executeWorkflowEntry(ctx context.Context, workflow Workflow, entryID string, input map[string]any, simulator Simulator, injectFault bool, now time.Time, trace *[]TraceEvent, stack map[string]bool, depth int) (map[string]any, error) {
	output, runErr := executeGraph(ctx, workflow, entryID, input, simulator, injectFault, now, trace, stack, depth)
	if runErr == nil || workflow.ErrorWorkflowID == "" {
		return output, runErr
	}
	handler := workflow.Subworkflows[workflow.ErrorWorkflowID]
	if handler == nil || handler.ErrorNodeID == "" {
		return output, fmt.Errorf("workflow %s failed: %w; configured error workflow %s is not executable", workflow.ID, runErr, workflow.ErrorWorkflowID)
	}
	errorInput := map[string]any{"error": map[string]any{"message": runErr.Error(), "workflowId": workflow.ID}, "execution": map[string]any{"status": "failed"}}
	handlerStack := cloneSet(stack)
	handlerStack[workflow.ID] = true
	if _, handlerErr := executeGraph(ctx, *handler, handler.ErrorNodeID, errorInput, simulator, false, now, trace, handlerStack, depth+1); handlerErr != nil {
		return output, fmt.Errorf("workflow %s failed: %w; error workflow %s failed: %v", workflow.ID, runErr, handler.ID, handlerErr)
	}
	return output, runErr
}

func executeGraph(ctx context.Context, workflow Workflow, entryID string, initial map[string]any, simulator Simulator, injectFault bool, now time.Time, trace *[]TraceEvent, stack map[string]bool, depth int) (map[string]any, error) {
	return executeGraphMany(ctx, workflow, entryID, []map[string]any{initial}, simulator, injectFault, now, trace, stack, depth, "", nil, nil)
}

func executeGraphMany(ctx context.Context, workflow Workflow, entryID string, initial []map[string]any, simulator Simulator, injectFault bool, now time.Time, trace *[]TraceEvent, stack map[string]bool, depth int, captureNode string, captured *[]map[string]any, seedNodeOutputs map[string][]map[string]any) (map[string]any, error) {
	if depth > maxSubworkflowDepth {
		return nil, fmt.Errorf("subworkflow nesting exceeds %d", maxSubworkflowDepth)
	}
	if workflow.ID != "" && stack[workflow.ID] {
		return nil, fmt.Errorf("recursive workflow reference detected at %s", workflow.ID)
	}
	stack = cloneSet(stack)
	stack[workflow.ID] = true
	nodes := workflow.NodesByID()
	if nodes[entryID].ID == "" {
		return nil, fmt.Errorf("workflow %s entry node %s is missing", workflow.ID, entryID)
	}
	order, err := topologicalOrder(workflow)
	if err != nil {
		return nil, err
	}
	initialItems := make([]map[string]any, 0, len(initial))
	for _, item := range initial {
		initialItems = append(initialItems, cloneMap(item))
	}
	inputs := map[string][]map[string]any{entryID: initialItems}
	inputsByPort := map[string]map[int][]map[string]any{entryID: {0: initialItems}}
	priorNodeOutputs := make(map[string][]map[string]any, len(seedNodeOutputs)+len(workflow.Nodes))
	for name, outputs := range seedNodeOutputs {
		priorNodeOutputs[name] = cloneMapSlice(outputs)
	}
	incomingSources := map[string]map[string]bool{}
	feedbackEdges, err := loopFeedbackEdges(workflow)
	if err != nil {
		return nil, err
	}
	for _, edge := range workflow.Connections {
		if feedbackEdges[edge] {
			continue
		}
		if incomingSources[edge.To] == nil {
			incomingSources[edge.To] = map[string]bool{}
		}
		incomingSources[edge.To][edge.From] = true
	}
	var last map[string]any
	for _, id := range order {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		node := nodes[id]
		var contexts []map[string]any
		if node.Trigger != "" {
			if node.ID != entryID {
				continue
			}
			contexts = cloneMapSlice(initialItems)
			priorNodeOutputs[node.Name] = cloneMapSlice(contexts)
		} else {
			incoming := inputs[id]
			if node.MergeUseDataOfInput > 0 {
				incoming = inputsByPort[id][node.MergeUseDataOfInput-1]
			}
			if node.ExecuteOnce && len(incoming) > 1 {
				incoming = incoming[:1]
			}
			if len(incoming) == 0 {
				continue
			}
			if node.MergeUseDataOfInput > 0 {
				contexts = incoming
			} else if len(incomingSources[id]) > 1 {
				contexts = []map[string]any{mergeMaps(incoming)}
			} else {
				contexts = incoming
			}
		}
		if node.LoopBatchSize > 0 {
			body, bodyEntry, err := linearLoopBody(workflow, node)
			if err != nil {
				return last, err
			}
			loopResults := make([]map[string]any, 0, len(contexts))
			batchSize := int(node.LoopBatchSize)
			for start := 0; start < len(contexts); start += batchSize {
				if err := ctx.Err(); err != nil {
					return last, err
				}
				end := start + batchSize
				if end > len(contexts) {
					end = len(contexts)
				}
				batch := contexts[start:end]
				// A deterministic virtual millisecond gives each invocation a stable
				// but distinct random stream without introducing wall-clock waits.
				batchNow := now.Add(time.Duration(start/batchSize) * time.Millisecond)
				*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/Loop Over Items", RequestSummary: node.Name, ResponseSummary: fmt.Sprintf("processing batch %d (%d item(s))", start/batchSize+1, len(batch)), Status: "succeeded", OccurredAt: batchNow.UTC()})
				bodyStack := cloneSet(stack)
				delete(bodyStack, workflow.ID)
				batchOutputs := []map[string]any{}
				batchOutput, err := executeGraphMany(ctx, body, bodyEntry, batch, simulator, injectFault, batchNow, trace, bodyStack, depth, loopFeedbackSource(workflow, node.ID), &batchOutputs, priorNodeOutputs)
				if err != nil {
					return last, fmt.Errorf("Loop Over Items %s batch %d: %w", node.Name, start/batchSize+1, err)
				}
				if len(batchOutputs) > 0 {
					loopResults = append(loopResults, batchOutputs...)
				} else if batchOutput != nil {
					loopResults = append(loopResults, batchOutput)
				}
			}
			if len(loopResults) > 0 {
				last = loopResults[len(loopResults)-1]
			}
			priorNodeOutputs[node.Name] = cloneMapSlice(loopResults)
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == 0 {
					for _, item := range loopResults {
						appendRoutedInput(inputs, inputsByPort, edge, item)
					}
				}
			}
			continue
		}
		if node.ExtractFromFilePropertyName != "" {
			outputs := make([]map[string]any, 0, len(contexts))
			for _, item := range contexts {
				binaryData, ok := item["binary"].(map[string]any)
				if !ok {
					return last, fmt.Errorf("Extract From File %s input has no binary object", node.Name)
				}
				file, ok := binaryData[node.ExtractFromFilePropertyName].(map[string]any)
				if !ok {
					return last, fmt.Errorf("Extract From File %s binary property %s is missing", node.Name, node.ExtractFromFilePropertyName)
				}
				encoded, ok := file["data"].(string)
				if !ok || len(encoded) > ((1<<20)*4/3+8) {
					return last, fmt.Errorf("Extract From File %s requires bounded base64 data", node.Name)
				}
				raw, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil || len(raw) > 1<<20 {
					return last, fmt.Errorf("Extract From File %s base64 payload is invalid or oversized", node.Name)
				}
				decoder := json.NewDecoder(strings.NewReader(string(raw)))
				decoder.UseNumber()
				var decoded any
				if err = decoder.Decode(&decoded); err != nil {
					return last, fmt.Errorf("Extract From File %s JSON is invalid: %w", node.Name, err)
				}
				if err = ensureJSONEnd(decoder); err != nil {
					return last, fmt.Errorf("Extract From File %s JSON has trailing data", node.Name)
				}
				value, err := toSafeJSON(decoded, 0)
				if err != nil {
					return last, fmt.Errorf("Extract From File %s JSON exceeds safe structure bounds: %w", node.Name, err)
				}
				out := cloneMap(item)
				out[node.ExtractFromFileDestination] = value
				outputs = append(outputs, out)
			}
			if len(outputs) > 0 {
				last = mergeMaps(outputs)
			}
			priorNodeOutputs[node.Name] = cloneMapSlice(outputs)
			if captured != nil && id == captureNode {
				*captured = cloneMapSlice(outputs)
			}
			*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/Extract bounded JSON file", RequestSummary: node.Name, ResponseSummary: fmt.Sprintf("decoded %d synthetic JSON files", len(outputs)), Status: "succeeded", OccurredAt: now.UTC()})
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == 0 {
					for _, item := range outputs {
						appendRoutedInput(inputs, inputsByPort, edge, item)
					}
				}
			}
			continue
		}
		if len(node.SortFields) > 0 {
			outputs := sortWorkflowItems(contexts, node.SortFields)
			*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/Sort bounded fields", RequestSummary: node.Name, ResponseSummary: fmt.Sprintf("sorted %d synthetic items", len(outputs)), Status: "succeeded", OccurredAt: now.UTC()})
			last = mergeMaps(outputs)
			priorNodeOutputs[node.Name] = cloneMapSlice(outputs)
			if captured != nil && id == captureNode {
				*captured = cloneMapSlice(outputs)
			}
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == 0 {
					for _, item := range outputs {
						appendRoutedInput(inputs, inputsByPort, edge, item)
					}
				}
			}
			continue
		}
		if len(node.SummarizeCountFields) > 0 {
			outputs, err := summarizeItems(contexts, node)
			if err != nil {
				return last, fmt.Errorf("Summarize node %s: %w", node.Name, err)
			}
			*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/Summarize grouped counts", RequestSummary: node.Name, ResponseSummary: fmt.Sprintf("summarized %d synthetic items into %d groups", len(contexts), len(outputs)), Status: "succeeded", OccurredAt: now.UTC()})
			last = mergeMaps(outputs)
			priorNodeOutputs[node.Name] = cloneMapSlice(outputs)
			if captured != nil && id == captureNode {
				*captured = cloneMapSlice(outputs)
			}
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == 0 {
					for _, item := range outputs {
						appendRoutedInput(inputs, inputsByPort, edge, item)
					}
				}
			}
			continue
		}
		if len(node.AggregateFields) > 0 {
			aggregated, err := aggregateItems(contexts, node)
			if err != nil {
				return last, fmt.Errorf("Aggregate node %s: %w", node.Name, err)
			}
			*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/Aggregate individual fields", RequestSummary: node.Name, ResponseSummary: fmt.Sprintf("aggregated %d synthetic items", len(contexts)), Status: "succeeded", OccurredAt: now.UTC()})
			last = aggregated
			priorNodeOutputs[node.Name] = []map[string]any{cloneMap(aggregated)}
			if captured != nil && id == captureNode {
				*captured = []map[string]any{cloneMap(aggregated)}
			}
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == 0 {
					appendRoutedInput(inputs, inputsByPort, edge, aggregated)
				}
			}
			continue
		}
		if node.SafeCode != "" {
			var codeOutput []map[string]any
			if node.SafeCodeRunOnceForEachItem {
				if len(contexts) > maxSafeCodeEachItems {
					return last, fmt.Errorf("Code node %s input exceeds the %d per-item execution bound", node.Name, maxSafeCodeEachItems)
				}
				for _, contextItem := range contexts {
					itemOutput, err := executeSafeCodePerItemWithNodeOutputs(node.SafeCode, contextItem, priorNodeOutputs, now)
					if err != nil {
						return last, fmt.Errorf("Code node %s rejected or failed in safe per-item interpreter: %w", node.Name, err)
					}
					codeOutput = append(codeOutput, itemOutput)
				}
			} else {
				runSafeCode := executeSafeCodeWithNodeOutputs
				if node.SafeCodeLegacyFunction {
					runSafeCode = executeSafeLegacyFunctionWithNodeOutputs
				}
				var err error
				codeOutput, err = runSafeCode(node.SafeCode, contexts, priorNodeOutputs, now)
				if err != nil {
					return last, fmt.Errorf("Code node %s rejected or failed in safe interpreter: %w", node.Name, err)
				}
			}
			outputs := make([]map[string]any, 0, len(codeOutput))
			for _, item := range codeOutput {
				payload, ok := item["json"].(map[string]any)
				if !ok {
					return last, fmt.Errorf("Code node %s returned an invalid item", node.Name)
				}
				outputs = append(outputs, cloneMap(payload))
			}
			if len(outputs) > 0 {
				last = mergeMaps(outputs)
			}
			priorNodeOutputs[node.Name] = cloneMapSlice(outputs)
			if captured != nil && id == captureNode {
				*captured = cloneMapSlice(outputs)
			}
			*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/Safe Code subset", RequestSummary: node.Name, ResponseSummary: fmt.Sprintf("safe Code interpreter emitted %d synthetic items", len(outputs)), Status: "succeeded", OccurredAt: now.UTC()})
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == 0 {
					for _, item := range outputs {
						appendRoutedInput(inputs, inputsByPort, edge, item)
					}
				}
			}
			continue
		}
		nodeOutputs := make([]map[string]any, 0)
		for _, input := range contexts {
			outputPort := 0
			output := cloneMap(input)
			outputs := []map[string]any{output}
			operation := node.Name
			responseSummary := "synthetic dataflow"
			switch {
			case node.LangChain != nil && node.LangChain.Kind == "pinecone_vector_store":
				chunks, payloadBytes, response, err := executePineconeInsert(ctx, workflow, node, priorNodeOutputs, simulator, injectFault, now, trace)
				if err != nil {
					return last, fmt.Errorf("LangChain Pinecone operation %s: %w", node.Name, err)
				}
				responseSummary = fmt.Sprintf("simulated Pinecone insert of %d chunk(s), %d-byte payload; %s", chunks, payloadBytes, response)
				operation = "LangChain Pinecone vector upsert"
			case node.WaitSeconds > 0 || node.WaitAmountPath != "":
				// Rehearsal uses virtual time: it proves the bounded continuation
				// edge without sleeping or persisting a deferred execution.
				seconds := node.WaitSeconds
				if node.WaitAmountPath != "" {
					amountValue, ok := jsonPathValue(input, node.WaitAmountPath)
					amount, number := numberValue(amountValue)
					multiplier := waitUnitSeconds(node.WaitUnit)
					maxSeconds := float64(30 * 24 * 60 * 60)
					if !ok || !number || amount < 1 || amount != float64(uint64(amount)) || multiplier == 0 || amount > maxSeconds/float64(multiplier) {
						return last, fmt.Errorf("Wait node %s amount path %s is missing or outside the bounded integer range", node.Name, node.WaitAmountPath)
					}
					seconds = uint32(amount * float64(multiplier))
				}
				responseSummary = fmt.Sprintf("virtual resume after %d seconds", seconds)
				operation = "Wait / virtual resume"
			case node.Request != nil:
				request := *node.Request
				if node.RSSURLPath != "" {
					feedValue, ok := jsonPathValue(input, node.RSSURLPath)
					feedURL, valid := feedValue.(string)
					parsed, parseErr := url.Parse(feedURL)
					if !ok || !valid || len(feedURL) > 2048 || parseErr != nil || parsed.Scheme != "https" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".synthetic.invalid") {
						return last, fmt.Errorf("RSS node %s requires a bounded HTTPS *.synthetic.invalid feed URL", node.Name)
					}
					request.Input = map[string]any{"feedUrl": feedURL}
					requestSummary := "bounded synthetic feed input"
					response, err := invokeWithRetry(ctx, simulator, request, injectFault, now, trace, workflow.ID, node.Name)
					if err != nil {
						return last, err
					}
					responseItems, err := decodeSimulatorItems(response)
					if err != nil {
						return last, fmt.Errorf("synthetic RSS response for %s: %w", node.Name, err)
					}
					outputs = make([]map[string]any, 0, len(responseItems))
					for _, responseItem := range responseItems {
						outputs = append(outputs, cloneMap(responseItem))
					}
					if len(outputs) > 0 {
						output = mergeMaps(outputs)
					}
					responseSummary = fmt.Sprintf("%s: %d deterministic synthetic articles", requestSummary, len(outputs))
					operation = "RSS Feed Read / synthetic profile"
				} else if node.RequestURLExpression != "" {
					request, err = requestWithSafeURLExpression(request, node.RequestURLExpression, priorNodeOutputs)
					if err != nil {
						return last, fmt.Errorf("resolve synthetic URL for %s: %w", node.Name, err)
					}
				}
				if node.RSSURLPath != "" {
					// The branch above already invoked the fixed synthetic RSS adapter.
				} else {
					response, err := invokeWithRetry(ctx, simulator, request, injectFault, now, trace, workflow.ID, node.Name)
					if err != nil {
						if !node.ErrorOutput {
							return last, err
						}
						outputPort = 1
						output = mergeMaps([]map[string]any{input, {"error": map[string]any{"message": "synthetic service operation failed"}}})
						outputs = []map[string]any{cloneMap(output)}
						responseSummary = "continued through synthetic error output"
					} else {
						responseItems, decodeErr := decodeSimulatorItems(response)
						if decodeErr != nil {
							return last, fmt.Errorf("simulated response for %s: %w", node.Name, decodeErr)
						}
						outputs = make([]map[string]any, 0, len(responseItems))
						for _, responseItem := range responseItems {
							if node.RequestReplacesInput {
								outputs = append(outputs, cloneMap(responseItem))
							} else {
								outputs = append(outputs, mergeMaps([]map[string]any{input, responseItem}))
							}
						}
						if len(outputs) > 0 {
							output = mergeMaps(outputs)
						}
					}
					operation = node.Request.Operation
				}
			case node.Condition != nil:
				matches := evaluateGraphCondition(*node.Condition, input, priorNodeOutputs, now)
				if !matches {
					outputPort = 1
				}
				operation = "If " + node.Condition.Path + " " + node.Condition.Operator
			case len(node.FilterConditions) > 0:
				matches := evaluateFilterConditions(node.FilterConditions, node.FilterCombinator, input, priorNodeOutputs, now)
				if !matches {
					outputPort = 1
					responseSummary = "sent item to discarded output"
				} else {
					responseSummary = "kept matching item"
				}
				operation = "Filter " + node.FilterConditions[0].Path + " " + node.FilterConditions[0].Operator
			case node.FilterCondition != nil:
				matches := evaluateGraphCondition(*node.FilterCondition, input, priorNodeOutputs, now)
				if !matches {
					outputPort = 1
					responseSummary = "sent item to discarded output"
				} else {
					responseSummary = "kept matching item"
				}
				operation = "Filter " + node.FilterCondition.Path + " " + node.FilterCondition.Operator
			case node.OutputPath != "":
				output = projectedValue(input, node.OutputPath)
				outputs = []map[string]any{output}
				operation = "Set JSON projection"
			case node.SetValues != nil:
				values := map[string]any{}
				for key, value := range node.SetValues {
					assignJSONPath(values, key, value)
				}
				for key, path := range node.SetProjections {
					if value, ok := jsonPathValue(input, path); ok {
						assignJSONPath(values, key, value)
					}
				}
				for key, expression := range node.SetExpressions {
					value, expressionErr := executeSafeExpressionWithNodeOutputs(expression, input, priorNodeOutputs, now)
					if expressionErr != nil {
						return last, fmt.Errorf("Set expression %s failed in safe interpreter: %w", key, expressionErr)
					}
					assignJSONPath(values, key, value)
				}
				if node.SetIncludeInput {
					output = mergeMaps([]map[string]any{input, values})
				} else {
					output = values
				}
				outputs = []map[string]any{output}
				operation = "Set bounded assignments"
			case node.SplitField != "":
				value, ok := jsonPathValue(input, node.SplitField)
				items, isArray := value.([]any)
				if !ok || !isArray || len(items) > 10000 {
					return last, fmt.Errorf("Split Out field %s is missing, not an array, or over the 10000-item bound", node.SplitField)
				}
				outputs = make([]map[string]any, 0, len(items))
				for _, item := range items {
					object, ok := item.(map[string]any)
					if !ok {
						return last, fmt.Errorf("Split Out field %s contains a non-object item", node.SplitField)
					}
					outputs = append(outputs, cloneMap(object))
				}
				responseSummary = fmt.Sprintf("split into %d synthetic items", len(outputs))
				operation = "Split Out " + node.SplitField
			case len(node.SwitchRules) > 0:
				outputPort = -1
				for _, rule := range node.SwitchRules {
					if evaluateGraphCondition(rule.Condition, input, priorNodeOutputs, now) {
						outputPort = rule.OutputPort
						break
					}
				}
				if outputPort < 0 && node.SwitchFallbackOutput {
					outputPort = len(node.SwitchRules)
				}
				responseSummary = fmt.Sprintf("selected output %d", outputPort)
				if status, ok := input["status"].(string); ok {
					responseSummary += " for status " + status
				}
				operation = "Switch"
			case node.SubworkflowID != "":
				child := workflow.Subworkflows[node.SubworkflowID]
				if child == nil {
					return last, fmt.Errorf("subworkflow %s is not in the import bundle", node.SubworkflowID)
				}
				entry := child.SubworkflowEntryID
				if entry == "" {
					return last, fmt.Errorf("subworkflow %s has no Execute Workflow Trigger", child.ID)
				}
				nested, err := executeWorkflowEntry(ctx, *child, entry, input, simulator, injectFault, now, trace, stack, depth+1)
				if err != nil {
					return last, err
				}
				output = mergeMaps([]map[string]any{input, nested})
				outputs = []map[string]any{output}
				operation = "Execute Workflow " + child.ID
			case node.Trigger == TriggerError:
				operation = "Error Trigger"
			case node.Trigger == TriggerSubworkflow:
				operation = "Execute Workflow Trigger"
			case node.Type == "n8n-nodes-base.respondToWebhook":
				operation = "Respond to Webhook"
			case node.Type == "n8n-nodes-base.stickyNote":
				operation = "Non-executable Note"
			case node.Trigger != "":
				operation = string(node.Trigger) + " Trigger"
			case node.Type == "n8n-nodes-base.noOp":
				operation = "No Operation"
			}
			if node.AlwaysOutputData && len(outputs) == 0 {
				outputs = []map[string]any{{}}
				output = outputs[0]
			}
			*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/" + operation, RequestSummary: node.Name, ResponseSummary: responseSummary, Status: "succeeded", OccurredAt: now.UTC()})
			last = output
			nodeOutputs = append(nodeOutputs, cloneMapSlice(outputs)...)
			if captured != nil && id == captureNode {
				*captured = cloneMapSlice(outputs)
			}
			for _, edge := range workflow.Connections {
				if edge.From == id && edge.OutputPort == outputPort {
					for _, item := range outputs {
						appendRoutedInput(inputs, inputsByPort, edge, item)
					}
				}
			}
		}
		priorNodeOutputs[node.Name] = nodeOutputs
	}
	return last, nil
}

func aggregateItems(items []map[string]any, node WorkflowNode) (map[string]any, error) {
	if len(items) > maxSyntheticItems {
		return nil, fmt.Errorf("input item count exceeds %d", maxSyntheticItems)
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxSafeCodeInputBytes {
		return nil, fmt.Errorf("input byte bound exceeded")
	}
	result := make(map[string]any, len(node.AggregateFields))
	for _, field := range node.AggregateFields {
		values := make([]any, 0, len(items))
		for _, item := range items {
			var value any
			var ok bool
			if node.AggregateDisableDotNotation {
				value, ok = item[field.InputPath]
			} else {
				value, ok = jsonPathValue(item, field.InputPath)
			}
			if !ok || value == nil {
				if node.AggregateKeepMissing {
					values = append(values, nil)
				}
				continue
			}
			if list, ok := value.([]any); ok && node.AggregateMergeLists {
				values = append(values, list...)
				if len(values) > maxSyntheticItems {
					return nil, fmt.Errorf("aggregated list exceeds item bound")
				}
			} else {
				values = append(values, value)
			}
			if len(values) > maxSyntheticItems {
				return nil, fmt.Errorf("aggregated field exceeds item bound")
			}
		}
		result[field.OutputName] = values
	}
	encoded, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxSafeCodeOutputBytes {
		return nil, fmt.Errorf("output byte bound exceeded")
	}
	return result, nil
}

func summarizeItems(items []map[string]any, node WorkflowNode) ([]map[string]any, error) {
	if len(items) > maxSyntheticItems || len(node.SummarizeCountFields) == 0 || len(node.SummarizeCountFields) > 32 {
		return nil, fmt.Errorf("input or summarize field count exceeds its bound")
	}
	groups := make(map[string]map[string]any)
	order := make([]string, 0, len(items))
	for _, item := range items {
		group := make(map[string]any, len(node.SummarizeGroupBy)+len(node.SummarizeCountFields))
		keyParts := make([]any, 0, len(node.SummarizeGroupBy))
		for _, path := range node.SummarizeGroupBy {
			value, ok := jsonPathValue(item, path)
			if !ok {
				value = nil
			}
			group[path] = value
			keyParts = append(keyParts, value)
		}
		encodedKey, err := json.Marshal(keyParts)
		if err != nil {
			return nil, err
		}
		key := string(encodedKey)
		acc, exists := groups[key]
		if !exists {
			acc = group
			for _, field := range node.SummarizeCountFields {
				acc["count_"+field] = float64(0)
			}
			groups[key] = acc
			order = append(order, key)
		}
		for _, field := range node.SummarizeCountFields {
			if value, ok := jsonPathValue(item, field); ok && value != nil {
				acc["count_"+field] = acc["count_"+field].(float64) + 1
			}
		}
	}
	result := make([]map[string]any, 0, len(order))
	for _, key := range order {
		result = append(result, groups[key])
	}
	return result, nil
}

func sortWorkflowItems(items []map[string]any, fields []SortField) []map[string]any {
	result := cloneMapSlice(items)
	if len(result) > maxSyntheticItems {
		return nil
	}
	sort.SliceStable(result, func(i, j int) bool {
		for _, field := range fields {
			left, leftOK := jsonPathValue(result[i], field.Path)
			right, rightOK := jsonPathValue(result[j], field.Path)
			comparison := compareSortValues(left, leftOK, right, rightOK)
			if comparison == 0 {
				continue
			}
			if field.Descending {
				return comparison > 0
			}
			return comparison < 0
		}
		return false
	})
	return result
}

func compareSortValues(left any, leftOK bool, right any, rightOK bool) int {
	if !leftOK && !rightOK {
		return 0
	}
	if !leftOK {
		return -1
	}
	if !rightOK {
		return 1
	}
	leftNumber, leftNumeric := left.(float64)
	rightNumber, rightNumeric := right.(float64)
	if leftNumeric && rightNumeric {
		switch {
		case leftNumber < rightNumber:
			return -1
		case leftNumber > rightNumber:
			return 1
		default:
			return 0
		}
	}
	leftText, rightText := fmt.Sprint(left), fmt.Sprint(right)
	if leftText < rightText {
		return -1
	}
	if leftText > rightText {
		return 1
	}
	return 0
}

func jsonPathValue(input map[string]any, path string) (any, bool) {
	var value any = input
	parts, ok := parseJSONPath(path)
	if !ok {
		return nil, false
	}
	for _, component := range parts {
		if component.isIndex {
			array, ok := value.([]any)
			if !ok || component.index >= len(array) {
				return nil, false
			}
			value = array[component.index]
			continue
		}
		object, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = object[component.key]
		if !ok {
			return nil, false
		}
	}
	return value, true
}

func assignJSONPath(target map[string]any, path string, value any) {
	segments, ok := parseJSONPath(path)
	if !ok || len(segments) == 0 {
		target[path] = value
		return
	}
	current := target
	for _, segment := range segments[:len(segments)-1] {
		if segment.isIndex {
			target[path] = value
			return
		}
		next, ok := current[segment.key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[segment.key] = next
		}
		current = next
	}
	last := segments[len(segments)-1]
	if last.isIndex {
		target[path] = value
		return
	}
	current[last.key] = value
}

func projectedValue(input map[string]any, path string) map[string]any {
	value, ok := jsonPathValue(input, path)
	if !ok {
		return map[string]any{}
	}
	if object, ok := value.(map[string]any); ok {
		return cloneMap(object)
	}
	return map[string]any{"value": value}
}

func invokeWithRetry(ctx context.Context, simulator Simulator, step Step, injectFault bool, now time.Time, trace *[]TraceEvent, workflowID, nodeName string) (string, error) {
	request := step.Method + " " + step.Host + step.URL
	for attempt := uint32(1); attempt <= 2; attempt++ {
		response, callErr := simulator.Invoke(ctx, step, attempt, injectFault)
		event := TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflowID + "/" + nodeName + ": " + step.Operation, RequestSummary: request, ResponseSummary: response, OccurredAt: now.UTC()}
		if callErr != nil {
			event.Status = "failed"
			event.ResponseSummary = callErr.Error()
			*trace = append(*trace, event)
			if attempt == 2 {
				return "", fmt.Errorf("operation %s failed after retry: %w", step.Operation, callErr)
			}
			continue
		}
		event.Status = "succeeded"
		*trace = append(*trace, event)
		return response, nil
	}
	return "", fmt.Errorf("operation %s did not produce a result", step.Operation)
}

func topologicalOrder(workflow Workflow) ([]string, error) {
	feedbackEdges, err := loopFeedbackEdges(workflow)
	if err != nil {
		return nil, err
	}
	indegree := make(map[string]int, len(workflow.Nodes))
	outgoing := make(map[string][]string, len(workflow.Nodes))
	for _, edge := range workflow.Connections {
		if feedbackEdges[edge] {
			continue
		}
		indegree[edge.To]++
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
	}
	orderIndex := make(map[string]int, len(workflow.Nodes))
	ready := make([]string, 0, len(workflow.Nodes))
	for index, node := range workflow.Nodes {
		orderIndex[node.ID] = index
		if indegree[node.ID] == 0 {
			ready = append(ready, node.ID)
		}
	}
	less := func(ids []string) {
		sort.Slice(ids, func(i, j int) bool { return orderIndex[ids[i]] < orderIndex[ids[j]] })
	}
	less(ready)
	order := make([]string, 0, len(workflow.Nodes))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, next := range outgoing[id] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				less(ready)
			}
		}
	}
	if len(order) != len(workflow.Nodes) {
		return nil, fmt.Errorf("workflow %s contains a graph cycle", workflow.ID)
	}
	return order, nil
}

func (workflow Workflow) NodesByID() map[string]WorkflowNode {
	result := make(map[string]WorkflowNode, len(workflow.Nodes))
	for _, node := range workflow.Nodes {
		result[node.ID] = node
	}
	return result
}

func syntheticTriggerInput(kind TriggerKind) map[string]any {
	action := "edited"
	if kind == TriggerWebhook {
		// A synthetic list event is a bounded, read-only sample for webhook APIs.
		action = "list"
	}
	input := map[string]any{"id": "synthetic-001", "status": "active", "source": string(kind), "email": "avery@example.invalid", "body": map[string]any{
		"action": action,
		"id":     "synthetic-001",
		"issue": map[string]any{
			"id":       float64(1),
			"title":    "Synthetic issue",
			"html_url": "https://github.com/synthetic-owner/synthetic-repo/issues/1",
		},
	}}
	if kind == TriggerPolling {
		// A generic row sample lets polling-trigger workflows exercise their
		// dataflow using synthetic values, independent of a particular template.
		input["Name"] = "Synthetic Lead"
		input["Email"] = "lead@example.invalid"
		input["Status"] = "New"
		input["Tag"] = "unprocessed"
	}
	return input
}

func evaluateCondition(condition Condition, input map[string]any) bool {
	if condition.Path == "$input.length" {
		// Nodes are invoked once per input item; an invocation proves the input
		// collection is nonempty. This is sufficient for comparisons against zero.
		left, leftErr := strconv.ParseFloat("1", 64)
		right, rightErr := strconv.ParseFloat(fmt.Sprint(condition.Value), 64)
		if leftErr != nil || rightErr != nil {
			return false
		}
		switch condition.Operator {
		case "gt":
			return left > right
		case "gte":
			return left >= right
		case "equals":
			return left == right
		}
		return false
	}
	value, ok := jsonPathValue(input, condition.Path)
	if !ok {
		return false
	}
	if condition.Operator == "numericRange" {
		bounds, valid := condition.Value.([]any)
		if !valid || len(bounds) != 2 {
			return false
		}
		actual, actualErr := strconv.ParseFloat(fmt.Sprint(value), 64)
		minimum, minErr := strconv.ParseFloat(fmt.Sprint(bounds[0]), 64)
		maximum, maxErr := strconv.ParseFloat(fmt.Sprint(bounds[1]), 64)
		return actualErr == nil && minErr == nil && maxErr == nil && actual > minimum && actual <= maximum
	}
	left := fmt.Sprint(value)
	right := fmt.Sprint(condition.Value)
	switch condition.Operator {
	case "true":
		return value == true
	case "false":
		return value == false
	case "equals":
		return left == right
	case "notEquals":
		return left != right
	case "contains":
		return strings.Contains(left, right)
	case "gt", "gte", "lt", "lte":
		l, lErr := strconv.ParseFloat(left, 64)
		r, rErr := strconv.ParseFloat(right, 64)
		if lErr != nil || rErr != nil {
			return false
		}
		switch condition.Operator {
		case "gt":
			return l > r
		case "gte":
			return l >= r
		case "lt":
			return l < r
		default:
			return l <= r
		}
	default:
		return false
	}
}

func evaluateGraphCondition(condition Condition, input map[string]any, prior map[string][]map[string]any, now time.Time) bool {
	if condition.Operator == "dateAfterNow" {
		left, ok := conditionValue(condition.NodeName, condition.Path, input, prior)
		if !ok {
			return false
		}
		leftTime, valid := parseWorkflowTime(left)
		return valid && leftTime.After(now.UTC())
	}
	if condition.NowFormat != "" {
		left, ok := conditionValue(condition.NodeName, condition.Path, input, prior)
		if !ok || condition.NowFormat != "2006-01-02" {
			return false
		}
		return fmt.Sprint(left) == now.UTC().Format(condition.NowFormat)
	}
	if condition.Operator == "dateAfterOrEquals" {
		left, ok := conditionValue(condition.NodeName, condition.Path, input, prior)
		if !ok {
			return false
		}
		base, ok := conditionValue(condition.RightNodeName, condition.RightPath, input, prior)
		if !ok {
			return false
		}
		duration, ok := conditionValue(condition.SubtractNodeName, condition.SubtractPath, input, prior)
		if !ok {
			return false
		}
		minutes, err := strconv.ParseFloat(fmt.Sprint(duration), 64)
		if err != nil || minutes < 0 || minutes > 30*24*60 {
			return false
		}
		leftTime, leftOK := parseWorkflowTime(left)
		baseTime, baseOK := parseWorkflowTime(base)
		return leftOK && baseOK && !leftTime.Before(baseTime.Add(-time.Duration(minutes*float64(time.Minute))))
	}
	if condition.NodeName == "" {
		return evaluateCondition(condition, input)
	}
	outputs, exists := prior[condition.NodeName]
	if !exists || len(outputs) == 0 {
		return false
	}
	return evaluateCondition(condition, outputs[0])
}

func evaluateFilterConditions(conditions []Condition, combinator string, input map[string]any, prior map[string][]map[string]any, now time.Time) bool {
	if len(conditions) == 0 || len(conditions) > 8 {
		return false
	}
	if combinator == "or" {
		for _, condition := range conditions {
			if evaluateGraphCondition(condition, input, prior, now) {
				return true
			}
		}
		return false
	}
	for _, condition := range conditions {
		if !evaluateGraphCondition(condition, input, prior, now) {
			return false
		}
	}
	return true
}

func conditionValue(nodeName, path string, input map[string]any, prior map[string][]map[string]any) (any, bool) {
	if nodeName == "" {
		return jsonPathValue(input, path)
	}
	outputs, ok := prior[nodeName]
	if !ok || len(outputs) == 0 {
		return nil, false
	}
	return jsonPathValue(outputs[0], path)
}

func parseWorkflowTime(value any) (time.Time, bool) {
	text := fmt.Sprint(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z07:00", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func mergeMaps(items []map[string]any) map[string]any {
	result := map[string]any{}
	for _, item := range items {
		for key, value := range item {
			result[key] = value
		}
	}
	return result
}

func appendRoutedInput(inputs map[string][]map[string]any, byPort map[string]map[int][]map[string]any, edge Connection, item map[string]any) {
	copy := cloneMap(item)
	inputs[edge.To] = append(inputs[edge.To], copy)
	if byPort[edge.To] == nil {
		byPort[edge.To] = map[int][]map[string]any{}
	}
	byPort[edge.To][edge.InputPort] = append(byPort[edge.To][edge.InputPort], cloneMap(item))
}

func cloneMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneMapSlice(input []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(input))
	for _, item := range input {
		result = append(result, cloneMap(item))
	}
	return result
}

func cloneSet(input map[string]bool) map[string]bool {
	result := make(map[string]bool, len(input)+1)
	for key, value := range input {
		result[key] = value
	}
	return result
}

func decodeObject(raw string) map[string]any {
	result := map[string]any{}
	if json.Unmarshal([]byte(raw), &result) != nil {
		return result
	}
	return result
}

// decodeSimulatorItems accepts the two generic n8n output shapes used by
// service profiles: a single JSON object or a bounded array of item objects.
func decodeSimulatorItems(raw string) ([]map[string]any, error) {
	if len(raw) > maxSafeCodeInputBytes {
		return nil, fmt.Errorf("response byte bound exceeded")
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("response must be JSON: %w", err)
	}
	switch typed := value.(type) {
	case map[string]any:
		return []map[string]any{typed}, nil
	case []any:
		if len(typed) > maxSyntheticItems {
			return nil, fmt.Errorf("response item count exceeds %d", maxSyntheticItems)
		}
		items := make([]map[string]any, 0, len(typed))
		for index, rawItem := range typed {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("response item %d must be an object", index)
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("response must be an object or array of objects")
	}
}
