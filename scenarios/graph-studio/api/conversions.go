package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConversionEngine handles graph format conversions
type ConversionEngine struct {
	converters map[string]map[string]Converter
}

// Converter defines the interface for format conversions
type Converter interface {
	Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error)
	GetMetadata() ConversionMetadata
}

// ConversionMetadata provides information about a conversion
type ConversionMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DataLoss    bool     `json:"data_loss"`      // Whether conversion may lose data
	Quality     string   `json:"quality"`        // high, medium, low
	Features    []string `json:"features"`       // List of supported features
}

// NewConversionEngine creates a new conversion engine
func NewConversionEngine() *ConversionEngine {
	engine := &ConversionEngine{
		converters: make(map[string]map[string]Converter),
	}
	engine.registerDefaultConverters()
	return engine
}

// registerDefaultConverters registers all built-in converters
func (ce *ConversionEngine) registerDefaultConverters() {
	// Mind Maps to other formats
	ce.RegisterConverter("mind-maps", "mermaid", &MindMapToMermaidConverter{})
	ce.RegisterConverter("mind-maps", "bpmn", &MindMapToBPMNConverter{})
	ce.RegisterConverter("mind-maps", "network-graphs", &MindMapToNetworkConverter{})
	
	// BPMN to other formats
	ce.RegisterConverter("bpmn", "mermaid", &BPMNToMermaidConverter{})
	ce.RegisterConverter("bpmn", "mind-maps", &BPMNToMindMapConverter{})
	ce.RegisterConverter("bpmn", "network-graphs", &BPMNToNetworkConverter{})
	
	// Network graphs to other formats
	ce.RegisterConverter("network-graphs", "mermaid", &NetworkToMermaidConverter{})
	ce.RegisterConverter("network-graphs", "mind-maps", &NetworkToMindMapConverter{})
	ce.RegisterConverter("network-graphs", "bpmn", &NetworkToBPMNConverter{})
	
	// Mermaid to other formats
	ce.RegisterConverter("mermaid", "mind-maps", &MermaidToMindMapConverter{})
	ce.RegisterConverter("mermaid", "bpmn", &MermaidToBPMNConverter{})
	ce.RegisterConverter("mermaid", "network-graphs", &MermaidToNetworkConverter{})
}

// RegisterConverter registers a new converter
func (ce *ConversionEngine) RegisterConverter(from, to string, converter Converter) {
	if ce.converters[from] == nil {
		ce.converters[from] = make(map[string]Converter)
	}
	ce.converters[from][to] = converter
}

// CanConvert checks if conversion between formats is supported
func (ce *ConversionEngine) CanConvert(from, to string) bool {
	return ce.converters[from] != nil && ce.converters[from][to] != nil
}

// Convert performs the actual conversion
func (ce *ConversionEngine) Convert(from, to string, data json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	if !ce.CanConvert(from, to) {
		return nil, fmt.Errorf("conversion from %s to %s not supported", from, to)
	}
	
	converter := ce.converters[from][to]
	return converter.Convert(data, options)
}

// GetSupportedConversions returns all supported conversion paths
func (ce *ConversionEngine) GetSupportedConversions() map[string][]string {
	result := make(map[string][]string)
	for from, targets := range ce.converters {
		for to := range targets {
			result[from] = append(result[from], to)
		}
	}
	return result
}

// GetConversionMetadata returns metadata about a specific conversion
func (ce *ConversionEngine) GetConversionMetadata(from, to string) (*ConversionMetadata, error) {
	if !ce.CanConvert(from, to) {
		return nil, fmt.Errorf("conversion from %s to %s not supported", from, to)
	}
	
	converter := ce.converters[from][to]
	metadata := converter.GetMetadata()
	return &metadata, nil
}

// Mind Maps to Mermaid Converter
type MindMapToMermaidConverter struct{}

func (c *MindMapToMermaidConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	var mindMap map[string]interface{}
	if err := json.Unmarshal(source, &mindMap); err != nil {
		return nil, fmt.Errorf("invalid mind map data: %w", err)
	}
	
	// Extract root node
	root, ok := mindMap["root"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mind map must have a root node")
	}
	
	// Build Mermaid diagram
	diagram := "graph TD\n"
	nodeCounter := 0
	nodeMap := make(map[string]string)
	
	// Process nodes recursively
	rootID := fmt.Sprintf("n%d", nodeCounter)
	nodeCounter++
	nodeMap["root"] = rootID
	
	if text, ok := root["text"].(string); ok {
		diagram += fmt.Sprintf("    %s[\"%s\"]\n", rootID, escapeMermaidText(text))
	}
	
	if children, ok := root["children"].([]interface{}); ok {
		diagram += c.processChildren(children, rootID, &nodeCounter, nodeMap)
	}
	
	result := map[string]interface{}{
		"type":    "mermaid",
		"diagram": diagram,
		"metadata": map[string]interface{}{
			"converted_from": "mind-maps",
			"node_count":     nodeCounter,
		},
	}
	
	return json.Marshal(result)
}

func (c *MindMapToMermaidConverter) processChildren(children []interface{}, parentID string, nodeCounter *int, nodeMap map[string]string) string {
	var result strings.Builder
	
	for _, child := range children {
		if childMap, ok := child.(map[string]interface{}); ok {
			childID := fmt.Sprintf("n%d", *nodeCounter)
			*nodeCounter++
			
			if text, ok := childMap["text"].(string); ok {
				result.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", childID, escapeMermaidText(text)))
				result.WriteString(fmt.Sprintf("    %s --> %s\n", parentID, childID))
			}
			
			if grandchildren, ok := childMap["children"].([]interface{}); ok {
				result.WriteString(c.processChildren(grandchildren, childID, nodeCounter, nodeMap))
			}
		}
	}
	
	return result.String()
}

func (c *MindMapToMermaidConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{
		Name:        "Mind Map to Mermaid",
		Description: "Converts hierarchical mind maps to Mermaid flowcharts",
		DataLoss:    false,
		Quality:     "high",
		Features:    []string{"hierarchy", "labels", "structure"},
	}
}

// BPMN to Mermaid Converter
type BPMNToMermaidConverter struct{}

func (c *BPMNToMermaidConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	var bpmn map[string]interface{}
	if err := json.Unmarshal(source, &bpmn); err != nil {
		return nil, fmt.Errorf("invalid BPMN data: %w", err)
	}
	
	diagram := "flowchart LR\n"
	
	// Process nodes
	if nodes, ok := bpmn["nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				id := nodeMap["id"].(string)
				label := nodeMap["label"].(string)
				nodeType := nodeMap["type"].(string)
				
				switch nodeType {
				case "startEvent":
					diagram += fmt.Sprintf("    %s(((\"%s\")))\n", id, escapeMermaidText(label))
				case "endEvent":
					diagram += fmt.Sprintf("    %s(((\"%s\")))\n", id, escapeMermaidText(label))
				case "task":
					diagram += fmt.Sprintf("    %s[\"%s\"]\n", id, escapeMermaidText(label))
				case "gateway":
					diagram += fmt.Sprintf("    %s{{\"%s\"}}\n", id, escapeMermaidText(label))
				default:
					diagram += fmt.Sprintf("    %s[\"%s\"]\n", id, escapeMermaidText(label))
				}
			}
		}
	}
	
	// Process edges
	if edges, ok := bpmn["edges"].([]interface{}); ok {
		for _, edge := range edges {
			if edgeMap, ok := edge.(map[string]interface{}); ok {
				source := edgeMap["source"].(string)
				target := edgeMap["target"].(string)
				diagram += fmt.Sprintf("    %s --> %s\n", source, target)
			}
		}
	}
	
	result := map[string]interface{}{
		"type":    "mermaid",
		"diagram": diagram,
		"metadata": map[string]interface{}{
			"converted_from": "bpmn",
			"process_type":   "workflow",
		},
	}
	
	return json.Marshal(result)
}

func (c *BPMNToMermaidConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{
		Name:        "BPMN to Mermaid",
		Description: "Converts BPMN process diagrams to Mermaid flowcharts",
		DataLoss:    false,
		Quality:     "high",
		Features:    []string{"processes", "gateways", "events", "tasks"},
	}
}

// Network to Mermaid Converter
type NetworkToMermaidConverter struct{}

func (c *NetworkToMermaidConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	var network map[string]interface{}
	if err := json.Unmarshal(source, &network); err != nil {
		return nil, fmt.Errorf("invalid network data: %w", err)
	}
	
	diagram := "graph TD\n"
	
	// Process nodes
	if nodes, ok := network["nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				id := nodeMap["id"].(string)
				label := nodeMap["label"].(string)
				
				diagram += fmt.Sprintf("    %s[\"%s\"]\n", id, escapeMermaidText(label))
			}
		}
	}
	
	// Process edges
	if edges, ok := network["edges"].([]interface{}); ok {
		for _, edge := range edges {
			if edgeMap, ok := edge.(map[string]interface{}); ok {
				source := edgeMap["source"].(string)
				target := edgeMap["target"].(string)
				
				// Check if it's bidirectional
				if bidirectional, ok := edgeMap["bidirectional"].(bool); ok && bidirectional {
					diagram += fmt.Sprintf("    %s <--> %s\n", source, target)
				} else {
					diagram += fmt.Sprintf("    %s --> %s\n", source, target)
				}
			}
		}
	}
	
	result := map[string]interface{}{
		"type":    "mermaid",
		"diagram": diagram,
		"metadata": map[string]interface{}{
			"converted_from": "network-graphs",
			"graph_type":     "network",
		},
	}
	
	return json.Marshal(result)
}

func (c *NetworkToMermaidConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{
		Name:        "Network to Mermaid",
		Description: "Converts network graphs to Mermaid diagrams",
		DataLoss:    false,
		Quality:     "high",
		Features:    []string{"nodes", "edges", "relationships"},
	}
}

// Placeholder converters for other combinations
type MindMapToBPMNConverter struct{}
func (c *MindMapToBPMNConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("bpmn", "mind-maps")
}
func (c *MindMapToBPMNConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Mind Map to BPMN", Description: "Converts mind maps to BPMN processes", DataLoss: true, Quality: "medium", Features: []string{"hierarchy to process"}}
}

type MindMapToNetworkConverter struct{}
func (c *MindMapToNetworkConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("network-graphs", "mind-maps")
}
func (c *MindMapToNetworkConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Mind Map to Network", Description: "Converts hierarchical mind maps to network graphs", DataLoss: false, Quality: "high", Features: []string{"hierarchy to network"}}
}

type BPMNToMindMapConverter struct{}
func (c *BPMNToMindMapConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("mind-maps", "bpmn")
}
func (c *BPMNToMindMapConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "BPMN to Mind Map", Description: "Converts BPMN processes to hierarchical mind maps", DataLoss: true, Quality: "medium", Features: []string{"process to hierarchy"}}
}

type BPMNToNetworkConverter struct{}
func (c *BPMNToNetworkConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("network-graphs", "bpmn")
}
func (c *BPMNToNetworkConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "BPMN to Network", Description: "Converts BPMN processes to network graphs", DataLoss: false, Quality: "high", Features: []string{"process to network"}}
}

type NetworkToMindMapConverter struct{}
func (c *NetworkToMindMapConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("mind-maps", "network-graphs")
}
func (c *NetworkToMindMapConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Network to Mind Map", Description: "Converts network graphs to hierarchical mind maps", DataLoss: true, Quality: "medium", Features: []string{"network to hierarchy"}}
}

type NetworkToBPMNConverter struct{}
func (c *NetworkToBPMNConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("bpmn", "network-graphs")
}
func (c *NetworkToBPMNConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Network to BPMN", Description: "Converts network graphs to BPMN processes", DataLoss: true, Quality: "medium", Features: []string{"network to process"}}
}

type MermaidToMindMapConverter struct{}
func (c *MermaidToMindMapConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("mind-maps", "mermaid")
}
func (c *MermaidToMindMapConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Mermaid to Mind Map", Description: "Converts Mermaid diagrams to mind maps", DataLoss: true, Quality: "medium", Features: []string{"diagram to hierarchy"}}
}

type MermaidToBPMNConverter struct{}
func (c *MermaidToBPMNConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("bpmn", "mermaid")
}
func (c *MermaidToBPMNConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Mermaid to BPMN", Description: "Converts Mermaid flowcharts to BPMN processes", DataLoss: true, Quality: "medium", Features: []string{"flowchart to process"}}
}

type MermaidToNetworkConverter struct{}
func (c *MermaidToNetworkConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return c.convertToPlaceholder("network-graphs", "mermaid")
}
func (c *MermaidToNetworkConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{Name: "Mermaid to Network", Description: "Converts Mermaid diagrams to network graphs", DataLoss: false, Quality: "high", Features: []string{"diagram to network"}}
}

// Helper functions

func (c *MindMapToBPMNConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"nodes": []map[string]interface{}{
			{"id": "placeholder", "label": "Converted Graph", "type": "placeholder"},
		},
	}
	return json.Marshal(result)
}

func (c *MindMapToNetworkConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"nodes": []map[string]interface{}{
			{"id": "placeholder", "label": "Converted Graph", "type": "placeholder"},
		},
	}
	return json.Marshal(result)
}

func (c *BPMNToMindMapConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"root": map[string]interface{}{
			"text": "Converted Graph",
			"children": []interface{}{},
		},
	}
	return json.Marshal(result)
}

func (c *BPMNToNetworkConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"nodes": []map[string]interface{}{
			{"id": "placeholder", "label": "Converted Graph", "type": "placeholder"},
		},
	}
	return json.Marshal(result)
}

func (c *NetworkToMindMapConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"root": map[string]interface{}{
			"text": "Converted Graph",
			"children": []interface{}{},
		},
	}
	return json.Marshal(result)
}

func (c *NetworkToBPMNConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"nodes": []map[string]interface{}{
			{"id": "placeholder", "label": "Converted Graph", "type": "placeholder"},
		},
	}
	return json.Marshal(result)
}

func (c *MermaidToMindMapConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"root": map[string]interface{}{
			"text": "Converted Graph",
			"children": []interface{}{},
		},
	}
	return json.Marshal(result)
}

func (c *MermaidToBPMNConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"nodes": []map[string]interface{}{
			{"id": "placeholder", "label": "Converted Graph", "type": "placeholder"},
		},
	}
	return json.Marshal(result)
}

func (c *MermaidToNetworkConverter) convertToPlaceholder(targetType, sourceType string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"type": targetType,
		"placeholder": true,
		"message": fmt.Sprintf("Conversion from %s to %s is not yet fully implemented", sourceType, targetType),
		"nodes": []map[string]interface{}{
			{"id": "placeholder", "label": "Converted Graph", "type": "placeholder"},
		},
	}
	return json.Marshal(result)
}

// escapeMermaidText escapes special characters for Mermaid
func escapeMermaidText(text string) string {
	text = strings.ReplaceAll(text, "\"", "\\\"")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}