package rehearsals

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestPublic20047ImportsLangChainAIDataflow(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/20047.json"))
	if err != nil {
		t.Fatalf("import public workflow 20047: %v", err)
	}
	kinds := map[string]string{}
	for _, node := range workflow.Nodes {
		if node.LangChain != nil {
			kinds[node.Name] = node.LangChain.Kind
		}
	}
	want := map[string]string{
		"Generate OpenAI Embeddings": "embeddings_openai",
		"Split Content Text":         "recursive_text_splitter",
		"Load Content Documents":     "default_data_loader",
		"Store Content in Pinecone":  "pinecone_vector_store",
	}
	for name, kind := range want {
		if kinds[name] != kind {
			t.Errorf("node %q normalized as %q, want %q", name, kinds[name], kind)
		}
	}
	if len(workflow.AIConnections) != 3 {
		t.Fatalf("AI connection graph has %d edges, want three: %#v", len(workflow.AIConnections), workflow.AIConnections)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-20047-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic WordPress, Data Table, and Pinecone services: %v", err)
	}
	if len(workflow.EntryNodeIDs) != 2 {
		t.Fatalf("public workflow 20047 has %d trigger entry points, want manual and schedule", len(workflow.EntryNodeIDs))
	}
	for _, entryID := range workflow.EntryNodeIDs {
		entry := workflow.NodesByID()[entryID]
		trace := []TraceEvent{}
		_, err = executeGraph(context.Background(), workflow, entryID, syntheticTriggerInput(entry.Trigger), simulator, false, time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), &trace, map[string]bool{}, 0)
		if err != nil {
			t.Fatalf("execute public workflow 20047 from %s against synthetic service descriptions: %v (trace=%#v)", entry.Trigger, err, trace)
		}
		if !traceHasSuccessfulOperation(trace, "@n8n/n8n-nodes-langchain.vectorStorePinecone:vector.upsert") {
			t.Fatalf("public workflow 20047 %s path did not use the synthetic Pinecone route: %#v", entry.Trigger, trace)
		}
	}
}

type capturingLangChainSimulator struct {
	steps []Step
}

func (sim *capturingLangChainSimulator) Invoke(_ context.Context, step Step, _ uint32, _ bool) (string, error) {
	sim.steps = append(sim.steps, step)
	return `{"upsertedCount":1}`, nil
}

func TestLangChainAttachmentFlowCreatesSyntheticPineconeVectors(t *testing.T) {
	workflowJSON := `{"id":"ai-flow","nodes":[{"id":"t","name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"format","name":"Format","type":"n8n-nodes-base.set","parameters":{"mode":"manual","includeOtherFields":false,"assignments":{"assignments":[{"name":"content","type":"string","value":"one two three four five six seven eight nine ten"},{"name":"title","type":"string","value":"Synthetic document"},{"name":"index","type":"string","value":"synthetic-index"},{"name":"namespace","type":"string","value":"demo"}]}}},{"id":"write","name":"Write","type":"n8n-nodes-base.noOp","parameters":{}},{"id":"embed","name":"Embeddings","type":"@n8n/n8n-nodes-langchain.embeddingsOpenAi","parameters":{"options":{}}},{"id":"split","name":"Split","type":"@n8n/n8n-nodes-langchain.textSplitterRecursiveCharacterTextSplitter","parameters":{"chunkSize":12,"chunkOverlap":3,"options":{}}},{"id":"loader","name":"Loader","type":"@n8n/n8n-nodes-langchain.documentDefaultDataLoader","parameters":{"jsonData":"={{ $('Format').item.json.content }}","jsonMode":"expressionData","textSplittingMode":"custom","options":{"metadata":{"metadataValues":[{"name":"title","value":"={{ $('Format').item.json.title }}"}]}}}},{"id":"store","name":"Store","type":"@n8n/n8n-nodes-langchain.vectorStorePinecone","parameters":{"mode":"insert","options":{"pineconeNamespace":"={{ $('Format').first().json.namespace }}"},"pineconeIndex":{"__rl":true,"mode":"id","value":"={{ $('Format').first().json.index }}"}}}],"connections":{"Start":{"main":[[{"node":"Format"}]]},"Format":{"main":[[{"node":"Write"}]]},"Write":{"main":[[{"node":"Store"}]]},"Split":{"ai_textSplitter":[[{"node":"Loader","type":"ai_textSplitter","index":0}]]},"Loader":{"ai_document":[[{"node":"Store","type":"ai_document","index":0}]]},"Embeddings":{"ai_embedding":[[{"node":"Store","type":"ai_embedding","index":0}]]}}}`
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize synthetic LangChain graph: %v", err)
	}
	simulator := &capturingLangChainSimulator{}
	trace := []TraceEvent{}
	output, err := executeGraph(context.Background(), workflow, workflow.EntryNodeIDs[0], map[string]any{"source": "manual"}, simulator, false, time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), &trace, map[string]bool{}, 0)
	if err != nil {
		t.Fatalf("execute synthetic LangChain graph: %v", err)
	}
	if output["content"] != "one two three four five six seven eight nine ten" {
		t.Fatalf("Pinecone operation did not preserve the main dataflow item: %#v", output)
	}
	if len(simulator.steps) != 1 || simulator.steps[0].NodeType != "@n8n/n8n-nodes-langchain.vectorStorePinecone" {
		t.Fatalf("expected one synthetic Pinecone API operation, got %#v", simulator.steps)
	}
	step := simulator.steps[0]
	if step.Host != "pinecone.synthetic.invalid" || step.URL != "/v1/vectors/upsert" || step.Action != "upsert" {
		t.Fatalf("Pinecone operation did not use its fixed synthetic route: %#v", step)
	}
	vectors, ok := step.Input["vectors"].([]any)
	if !ok || len(vectors) < 2 {
		t.Fatalf("recursive splitter did not produce overlapping bounded chunks: %#v", step.Input["vectors"])
	}
	if step.Input["index"] != "synthetic-index" || step.Input["namespace"] != "demo" {
		t.Fatalf("Pinecone selectors did not resolve from the upstream n8n item: %#v", step.Input)
	}
	firstVector, ok := vectors[0].(map[string]any)
	metadata, metadataOK := firstVector["metadata"].(map[string]any)
	values, valuesOK := firstVector["values"].([]any)
	if !ok || !metadataOK || !valuesOK || metadata["title"] != "Synthetic document" || len(values) != 16 || metadata["text"] == "" {
		t.Fatalf("Pinecone payload omitted deterministic synthetic embedding or document metadata: %#v", firstVector)
	}
	for _, name := range []string{"Split", "Loader", "Embeddings"} {
		found := false
		for _, event := range trace {
			if event.RequestSummary == name && event.Status == "succeeded" {
				found = true
			}
		}
		if !found {
			t.Errorf("trace omitted successful LangChain node %q: %#v", name, trace)
		}
	}
	encoded, err := json.Marshal(step.Input)
	if err != nil || len(encoded) == 0 || len(encoded) > maxSyntheticVectorBytes {
		t.Fatalf("Pinecone vector request has invalid bounded JSON: %d bytes, %v", len(encoded), err)
	}
}

func TestRecursiveCharacterChunksRespectRuneBoundsAndOverlap(t *testing.T) {
	chunks := recursiveCharacterChunks("αβγ δζη θικ λμν ξοπ", 7, 2)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %#v", chunks)
	}
	for i, chunk := range chunks {
		if len([]rune(chunk)) > 7 {
			t.Errorf("chunk %d exceeds rune bound: %q", i, chunk)
		}
	}
	if chunks[0] == chunks[1] {
		t.Errorf("expected overlap windows to advance: %#v", chunks)
	}
}

func TestLangChainInvalidAttachmentPairRejected(t *testing.T) {
	workflowJSON := `{"id":"bad","nodes":[{"id":"t","name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"s","name":"Set","type":"n8n-nodes-base.set","parameters":{"mode":"manual","assignments":{"assignments":[{"name":"x","type":"string","value":"x"}]}}},{"id":"e","name":"Embeddings","type":"@n8n/n8n-nodes-langchain.embeddingsOpenAi","parameters":{"options":{}}},{"id":"v","name":"Store","type":"@n8n/n8n-nodes-langchain.vectorStorePinecone","parameters":{"mode":"insert","pineconeIndex":{"mode":"id","value":"idx"},"options":{"pineconeNamespace":"ns"}}}],"connections":{"Start":{"main":[[{"node":"Set"}]]},"Set":{"main":[[{"node":"Store"}]]},"Embeddings":{"ai_document":[[{"node":"Store","type":"ai_document","index":0}]]}}}`
	if _, _, _, err := normalizeWorkflow(workflowJSON); err == nil {
		t.Fatal("expected a mismatched LangChain attachment type to be rejected")
	}
}

func Example_recursiveCharacterChunks() {
	chunks := recursiveCharacterChunks("a b c d e f", 4, 2)
	encoded, _ := json.Marshal(chunks)
	fmt.Println(string(encoded))
	// Output: ["a b ","b c ","c d ","d e ","e f"]
}
