package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxSyntheticDocumentBytes = 256 * 1024
	maxSyntheticChunks        = 1000
	maxSyntheticVectorBytes   = 512 * 1024
)

// executePineconeInsert interprets the typed LangChain attachment graph with
// deterministic synthetic text splitting and embeddings. The only external
// effect is a fixed vector-upsert operation through the injected simulator.
func executePineconeInsert(ctx context.Context, workflow Workflow, node WorkflowNode, prior map[string][]map[string]any, simulator Simulator, injectFault bool, now time.Time, trace *[]TraceEvent) (int, int, string, error) {
	if node.LangChain == nil || node.LangChain.Kind != "pinecone_vector_store" {
		return 0, 0, "", fmt.Errorf("node is not a Pinecone vector store")
	}
	var documentNode, embeddingNode string
	for _, edge := range workflow.AIConnections {
		if edge.To != node.ID {
			continue
		}
		switch edge.Type {
		case "ai_document":
			if documentNode != "" {
				return 0, 0, "", fmt.Errorf("Pinecone node %s has multiple document inputs", node.Name)
			}
			documentNode = edge.From
		case "ai_embedding":
			if embeddingNode != "" {
				return 0, 0, "", fmt.Errorf("Pinecone node %s has multiple embedding inputs", node.Name)
			}
			embeddingNode = edge.From
		}
	}
	nodes := workflow.NodesByID()
	loader := nodes[documentNode]
	embeddings := nodes[embeddingNode]
	if loader.LangChain == nil || loader.LangChain.Kind != "default_data_loader" || embeddings.LangChain == nil || embeddings.LangChain.Kind != "embeddings_openai" {
		return 0, 0, "", fmt.Errorf("Pinecone node %s requires one Default Data Loader and one OpenAI Embeddings input", node.Name)
	}
	var splitter WorkflowNode
	for _, edge := range workflow.AIConnections {
		if edge.To == loader.ID && edge.Type == "ai_textSplitter" {
			if splitter.ID != "" {
				return 0, 0, "", fmt.Errorf("Default Data Loader %s has multiple text splitters", loader.Name)
			}
			splitter = nodes[edge.From]
		}
	}
	if splitter.LangChain == nil || splitter.LangChain.Kind != "recursive_text_splitter" || loader.LangChain.DataReference == nil {
		return 0, 0, "", fmt.Errorf("Default Data Loader %s requires one recursive text splitter and a data reference", loader.Name)
	}
	documentValue, ok := resolveNodeValue(*loader.LangChain.DataReference, prior)
	if !ok {
		return 0, 0, "", fmt.Errorf("Default Data Loader %s data reference is unavailable", loader.Name)
	}
	document, ok := documentValue.(string)
	if !ok || !utf8.ValidString(document) || len(document) > maxSyntheticDocumentBytes {
		return 0, 0, "", fmt.Errorf("Default Data Loader %s requires UTF-8 text within %d bytes", loader.Name, maxSyntheticDocumentBytes)
	}
	metadata := make(map[string]any, len(loader.LangChain.MetadataRefs))
	for key, ref := range loader.LangChain.MetadataRefs {
		value, found := resolveNodeValue(ref, prior)
		if !found {
			return 0, 0, "", fmt.Errorf("Default Data Loader %s metadata reference %s is unavailable", loader.Name, key)
		}
		metadata[key] = value
	}
	chunks := recursiveCharacterChunks(document, splitter.LangChain.ChunkSize, splitter.LangChain.ChunkOverlap)
	if len(chunks) > maxSyntheticChunks {
		return 0, 0, "", fmt.Errorf("Default Data Loader %s generated more than %d chunks", loader.Name, maxSyntheticChunks)
	}
	index, ok := resolveNodeValue(*node.LangChain.Index, prior)
	if !ok {
		return 0, 0, "", fmt.Errorf("Pinecone node %s index selector is unavailable", node.Name)
	}
	namespace, ok := resolveNodeValue(*node.LangChain.Namespace, prior)
	if !ok {
		return 0, 0, "", fmt.Errorf("Pinecone node %s namespace selector is unavailable", node.Name)
	}
	indexName, indexOK := index.(string)
	namespaceName, namespaceOK := namespace.(string)
	if !indexOK || indexName == "" || len(indexName) > 128 || !namespaceOK || len(namespaceName) > 128 || strings.ContainsAny(indexName+namespaceName, "\x00\r\n") {
		return 0, 0, "", fmt.Errorf("Pinecone node %s index or namespace is not a bounded string", node.Name)
	}
	vectors := make([]any, 0, len(chunks))
	for i, chunk := range chunks {
		digest := sha256.Sum256([]byte(chunk))
		values := make([]any, 16)
		for j := range values {
			// The synthetic embedding is a stable hash projection, not a model result.
			values[j] = float64(digest[j]) / 255.0
		}
		vectorMetadata := cloneMapAny(metadata)
		vectorMetadata["text"] = chunk
		vectors = append(vectors, map[string]any{"id": hex.EncodeToString(digest[:12]) + fmt.Sprintf("-%04d", i), "values": values, "metadata": vectorMetadata})
	}
	inputData := map[string]any{"index": indexName, "namespace": namespaceName, "vectors": vectors}
	encoded, err := json.Marshal(inputData)
	if err != nil || len(encoded) > maxSyntheticVectorBytes {
		return 0, 0, "", fmt.Errorf("Pinecone node %s vector payload exceeds the synthetic %d-byte bound", node.Name, maxSyntheticVectorBytes)
	}
	appendLangChainTrace(trace, workflow.ID, splitter.Name, fmt.Sprintf("recursive splitter emitted %d bounded chunk(s)", len(chunks)), now)
	appendLangChainTrace(trace, workflow.ID, loader.Name, fmt.Sprintf("default loader created %d document(s) with %d metadata field(s)", len(chunks), len(metadata)), now)
	appendLangChainTrace(trace, workflow.ID, embeddings.Name, fmt.Sprintf("synthetic deterministic embedding produced for %d chunk(s)", len(chunks)), now)
	step := Step{ID: node.ID, Operation: node.Type + ":vector.upsert", NodeType: node.Type, Resource: "vector", Action: "upsert", Method: "POST", Host: "pinecone.synthetic.invalid", URL: "/v1/vectors/upsert", Input: inputData}
	response, err := invokeWithRetry(ctx, simulator, step, injectFault, now, trace, workflow.ID, node.Name)
	if err != nil {
		return 0, 0, "", err
	}
	return len(chunks), len(encoded), response, nil
}

func appendLangChainTrace(trace *[]TraceEvent, workflowID, nodeName, summary string, now time.Time) {
	*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflowID + "/LangChain synthetic", RequestSummary: nodeName, ResponseSummary: summary, Status: "succeeded", OccurredAt: now.UTC()})
}

func resolveNodeValue(ref NodeValueRef, prior map[string][]map[string]any) (any, bool) {
	if ref.NodeName == "" {
		return ref.Literal, true
	}
	items := prior[ref.NodeName]
	if len(items) == 0 {
		return nil, false
	}
	return jsonPathValue(items[0], ref.Path)
}

func recursiveCharacterChunks(text string, size, overlap int) []string {
	if text == "" {
		return nil
	}
	if size < 1 || overlap < 0 || overlap >= size {
		return nil
	}
	return splitRecursive(text, []string{"\n\n", "\n", " ", ""}, size, overlap)
}

func splitRecursive(text string, separators []string, size, overlap int) []string {
	if runeLen(text) <= size {
		return []string{text}
	}
	selected := len(separators) - 1
	for i, separator := range separators {
		if separator == "" || strings.Contains(text, separator) {
			selected = i
			break
		}
	}
	separator := separators[selected]
	pieces := splitKeepingSeparator(text, separator)
	short := make([]string, 0, len(pieces))
	chunks := make([]string, 0, (runeLen(text)+size-1)/size)
	flush := func() {
		if len(short) != 0 {
			chunks = append(chunks, mergeRecursiveSplits(short, "", size, overlap)...)
			short = short[:0]
		}
	}
	for _, piece := range pieces {
		if piece == "" {
			continue
		}
		if runeLen(piece) <= size {
			short = append(short, piece)
			continue
		}
		flush()
		if selected+1 < len(separators) {
			chunks = append(chunks, splitRecursive(piece, separators[selected+1:], size, overlap)...)
		} else {
			chunks = append(chunks, mergeRecursiveSplits(splitKeepingSeparator(piece, ""), "", size, overlap)...)
		}
	}
	flush()
	return chunks
}

func splitKeepingSeparator(text, separator string) []string {
	if separator == "" {
		runes := []rune(text)
		pieces := make([]string, len(runes))
		for i, char := range runes {
			pieces[i] = string(char)
		}
		return pieces
	}
	parts := strings.Split(text, separator)
	pieces := make([]string, 0, len(parts))
	for i, part := range parts {
		if i < len(parts)-1 {
			part += separator
		}
		if part != "" {
			pieces = append(pieces, part)
		}
	}
	return pieces
}

func mergeRecursiveSplits(pieces []string, separator string, size, overlap int) []string {
	chunks := make([]string, 0)
	current := make([]string, 0)
	for _, piece := range pieces {
		if piece == "" {
			continue
		}
		candidate := append(append([]string(nil), current...), piece)
		if len(current) > 0 && runeLen(strings.Join(candidate, separator)) > size {
			chunks = append(chunks, strings.Join(current, separator))
			for len(current) > 0 && (runeLen(strings.Join(current, separator)) > overlap || runeLen(strings.Join(append(append([]string(nil), current...), piece), separator)) > size) {
				current = current[1:]
			}
		}
		current = append(current, piece)
	}
	if len(current) > 0 {
		chunks = append(chunks, strings.Join(current, separator))
	}
	return chunks
}

func runeLen(value string) int { return utf8.RuneCountInString(value) }

func cloneMapAny(source map[string]any) map[string]any {
	copy := make(map[string]any, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
