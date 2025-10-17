package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// GraphMLExporter handles GraphML format export
type GraphMLExporter struct{}

// GEXFExporter handles GEXF format export
type GEXFExporter struct{}

// GraphML XML structures
type GraphML struct {
	XMLName           xml.Name     `xml:"graphml"`
	Xmlns             string       `xml:"xmlns,attr"`
	XmlnsXSI          string       `xml:"xmlns:xsi,attr"`
	XSISchemaLocation string       `xml:"xsi:schemaLocation,attr"`
	Keys              []GraphMLKey `xml:"key"`
	Graph             GraphMLGraph `xml:"graph"`
}

type GraphMLKey struct {
	ID       string `xml:"id,attr"`
	For      string `xml:"for,attr"`
	AttrName string `xml:"attr.name,attr"`
	AttrType string `xml:"attr.type,attr"`
}

type GraphMLGraph struct {
	ID          string        `xml:"id,attr"`
	EdgeDefault string        `xml:"edgedefault,attr"`
	Nodes       []GraphMLNode `xml:"node"`
	Edges       []GraphMLEdge `xml:"edge"`
}

type GraphMLNode struct {
	ID   string        `xml:"id,attr"`
	Data []GraphMLData `xml:"data"`
}

type GraphMLEdge struct {
	ID     string        `xml:"id,attr"`
	Source string        `xml:"source,attr"`
	Target string        `xml:"target,attr"`
	Data   []GraphMLData `xml:"data"`
}

type GraphMLData struct {
	Key   string `xml:"key,attr"`
	Value string `xml:",chardata"`
}

// GEXF XML structures
type GEXF struct {
	XMLName xml.Name  `xml:"gexf"`
	Xmlns   string    `xml:"xmlns,attr"`
	Version string    `xml:"version,attr"`
	Meta    GEXFMeta  `xml:"meta"`
	Graph   GEXFGraph `xml:"graph"`
}

type GEXFMeta struct {
	LastModified string `xml:"lastmodifieddate,attr"`
	Creator      string `xml:"creator"`
	Description  string `xml:"description"`
}

type GEXFGraph struct {
	Mode            string    `xml:"mode,attr"`
	DefaultEdgeType string    `xml:"defaultedgetype,attr"`
	Nodes           GEXFNodes `xml:"nodes"`
	Edges           GEXFEdges `xml:"edges"`
}

type GEXFNodes struct {
	Count int        `xml:"count,attr"`
	Nodes []GEXFNode `xml:"node"`
}

type GEXFNode struct {
	ID    string `xml:"id,attr"`
	Label string `xml:"label,attr"`
}

type GEXFEdges struct {
	Count int        `xml:"count,attr"`
	Edges []GEXFEdge `xml:"edge"`
}

type GEXFEdge struct {
	ID     string `xml:"id,attr"`
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
	Label  string `xml:"label,attr,omitempty"`
}

// ExportToGraphML exports a graph to GraphML format
func (e *GraphMLExporter) Export(graph *Graph) (string, error) {
	// Parse graph data
	var graphData map[string]interface{}
	if err := json.Unmarshal(graph.Data, &graphData); err != nil {
		return "", fmt.Errorf("failed to parse graph data: %w", err)
	}

	// Extract nodes and edges based on graph type
	nodes, edges := extractNodesAndEdges(graphData, graph.Type)

	// Build GraphML structure
	graphML := GraphML{
		Xmlns:             "http://graphml.graphdrawing.org/xmlns",
		XmlnsXSI:          "http://www.w3.org/2001/XMLSchema-instance",
		XSISchemaLocation: "http://graphml.graphdrawing.org/xmlns http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd",
		Keys: []GraphMLKey{
			{ID: "label", For: "node", AttrName: "label", AttrType: "string"},
			{ID: "type", For: "node", AttrName: "type", AttrType: "string"},
			{ID: "weight", For: "edge", AttrName: "weight", AttrType: "double"},
		},
		Graph: GraphMLGraph{
			ID:          graph.ID,
			EdgeDefault: "directed",
			Nodes:       make([]GraphMLNode, 0),
			Edges:       make([]GraphMLEdge, 0),
		},
	}

	// Add nodes
	for _, node := range nodes {
		graphMLNode := GraphMLNode{
			ID:   fmt.Sprintf("%v", node["id"]),
			Data: []GraphMLData{},
		}

		if label, ok := node["label"].(string); ok {
			graphMLNode.Data = append(graphMLNode.Data, GraphMLData{
				Key:   "label",
				Value: label,
			})
		}

		if nodeType, ok := node["type"].(string); ok {
			graphMLNode.Data = append(graphMLNode.Data, GraphMLData{
				Key:   "type",
				Value: nodeType,
			})
		}

		graphML.Graph.Nodes = append(graphML.Graph.Nodes, graphMLNode)
	}

	// Add edges
	for i, edge := range edges {
		graphMLEdge := GraphMLEdge{
			ID:     fmt.Sprintf("e%d", i),
			Source: fmt.Sprintf("%v", edge["source"]),
			Target: fmt.Sprintf("%v", edge["target"]),
			Data:   []GraphMLData{},
		}

		if weight, ok := edge["weight"].(float64); ok {
			graphMLEdge.Data = append(graphMLEdge.Data, GraphMLData{
				Key:   "weight",
				Value: fmt.Sprintf("%.2f", weight),
			})
		}

		graphML.Graph.Edges = append(graphML.Graph.Edges, graphMLEdge)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(graphML, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal GraphML: %w", err)
	}

	return xml.Header + string(output), nil
}

// ExportToGEXF exports a graph to GEXF format
func (e *GEXFExporter) Export(graph *Graph) (string, error) {
	// Parse graph data
	var graphData map[string]interface{}
	if err := json.Unmarshal(graph.Data, &graphData); err != nil {
		return "", fmt.Errorf("failed to parse graph data: %w", err)
	}

	// Extract nodes and edges based on graph type
	nodes, edges := extractNodesAndEdges(graphData, graph.Type)

	// Build GEXF structure
	gexf := GEXF{
		Xmlns:   "http://www.gexf.net/1.2draft",
		Version: "1.2",
		Meta: GEXFMeta{
			LastModified: time.Now().Format("2006-01-02"),
			Creator:      "Graph Studio",
			Description:  graph.Description,
		},
		Graph: GEXFGraph{
			Mode:            "static",
			DefaultEdgeType: "directed",
			Nodes: GEXFNodes{
				Count: len(nodes),
				Nodes: make([]GEXFNode, 0),
			},
			Edges: GEXFEdges{
				Count: len(edges),
				Edges: make([]GEXFEdge, 0),
			},
		},
	}

	// Add nodes
	for _, node := range nodes {
		gexfNode := GEXFNode{
			ID: fmt.Sprintf("%v", node["id"]),
		}

		if label, ok := node["label"].(string); ok {
			gexfNode.Label = label
		} else if text, ok := node["text"].(string); ok {
			gexfNode.Label = text
		} else {
			gexfNode.Label = gexfNode.ID
		}

		gexf.Graph.Nodes.Nodes = append(gexf.Graph.Nodes.Nodes, gexfNode)
	}

	// Add edges
	for i, edge := range edges {
		gexfEdge := GEXFEdge{
			ID:     fmt.Sprintf("e%d", i),
			Source: fmt.Sprintf("%v", edge["source"]),
			Target: fmt.Sprintf("%v", edge["target"]),
		}

		if label, ok := edge["label"].(string); ok {
			gexfEdge.Label = label
		}

		gexf.Graph.Edges.Edges = append(gexf.Graph.Edges.Edges, gexfEdge)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(gexf, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal GEXF: %w", err)
	}

	return xml.Header + string(output), nil
}

// extractNodesAndEdges extracts nodes and edges from graph data based on type
func extractNodesAndEdges(data map[string]interface{}, graphType string) ([]map[string]interface{}, []map[string]interface{}) {
	nodes := make([]map[string]interface{}, 0)
	edges := make([]map[string]interface{}, 0)

	switch graphType {
	case "network-graphs", "bpmn":
		// Standard nodes/edges structure
		if nodesData, ok := data["nodes"].([]interface{}); ok {
			for _, n := range nodesData {
				if node, ok := n.(map[string]interface{}); ok {
					nodes = append(nodes, node)
				}
			}
		}
		if edgesData, ok := data["edges"].([]interface{}); ok {
			for _, e := range edgesData {
				if edge, ok := e.(map[string]interface{}); ok {
					edges = append(edges, edge)
				}
			}
		}

	case "mind-maps":
		// Convert hierarchical structure to nodes/edges
		if root, ok := data["root"].(map[string]interface{}); ok {
			nodeID := 0
			var traverse func(node map[string]interface{}, parentID string) string
			traverse = func(node map[string]interface{}, parentID string) string {
				currentID := fmt.Sprintf("n%d", nodeID)
				nodeID++

				// Add node
				nodeData := map[string]interface{}{
					"id": currentID,
				}
				if text, ok := node["text"].(string); ok {
					nodeData["label"] = text
				}
				if nodeType, ok := node["type"].(string); ok {
					nodeData["type"] = nodeType
				}
				nodes = append(nodes, nodeData)

				// Add edge from parent
				if parentID != "" {
					edges = append(edges, map[string]interface{}{
						"source": parentID,
						"target": currentID,
					})
				}

				// Process children
				if children, ok := node["children"].([]interface{}); ok {
					for _, child := range children {
						if childMap, ok := child.(map[string]interface{}); ok {
							traverse(childMap, currentID)
						}
					}
				}

				return currentID
			}
			traverse(root, "")
		}

	case "mermaid":
		// Parse mermaid text to extract basic structure
		if mermaidText, ok := data["text"].(string); ok {
			// Simple parser for mermaid syntax
			lines := strings.Split(mermaidText, "\n")
			nodeMap := make(map[string]bool)

			for _, line := range lines {
				line = strings.TrimSpace(line)

				// Detect node definitions: A[Label] or A((Label)) etc
				if strings.Contains(line, "[") || strings.Contains(line, "((") {
					// Extract node ID
					parts := strings.FieldsFunc(line, func(r rune) bool {
						return r == '[' || r == '(' || r == '{' || r == '>' || r == ' '
					})
					if len(parts) > 0 {
						nodeID := parts[0]
						if !nodeMap[nodeID] {
							nodeMap[nodeID] = true
							nodes = append(nodes, map[string]interface{}{
								"id":    nodeID,
								"label": nodeID,
							})
						}
					}
				}

				// Detect edges: A --> B or A --> B
				if strings.Contains(line, "-->") || strings.Contains(line, "---") {
					parts := strings.Split(line, "-->")
					if len(parts) < 2 {
						parts = strings.Split(line, "---")
					}
					if len(parts) == 2 {
						source := strings.TrimSpace(parts[0])
						target := strings.TrimSpace(parts[1])

						// Clean up node IDs
						source = strings.FieldsFunc(source, func(r rune) bool {
							return r == '[' || r == '(' || r == ' '
						})[0]
						target = strings.FieldsFunc(target, func(r rune) bool {
							return r == '[' || r == '(' || r == ' '
						})[0]

						edges = append(edges, map[string]interface{}{
							"source": source,
							"target": target,
						})
					}
				}
			}
		}
	}

	return nodes, edges
}
