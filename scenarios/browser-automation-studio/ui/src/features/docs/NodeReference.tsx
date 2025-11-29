import { useState, useMemo } from "react";
import {
  Search,
  ChevronRight,
  ChevronDown,
  Globe,
  MousePointer,
  Keyboard,
  Database,
  ShieldCheck,
  GitBranch,
  Server,
} from "lucide-react";
import { MarkdownRenderer } from "./MarkdownRenderer";
import {
  NODE_DOCUMENTATION,
  CATEGORY_ORDER,
  getNodeDocumentationByCategory,
  type NodeDocEntry,
} from "./content/nodeDocumentation";
import { WORKFLOW_NODE_DEFINITIONS } from "@constants/nodeCategories";

interface NodeReferenceProps {
  initialNodeType?: string;
  onSelectNode?: (nodeType: string) => void;
}

const CATEGORY_ICONS: Record<string, React.ElementType> = {
  "Navigation & Context": Globe,
  "Pointer & Gestures": MousePointer,
  "Forms & Input": Keyboard,
  "Data & Variables": Database,
  "Assertions & Observability": ShieldCheck,
  "Workflow Logic": GitBranch,
  "Storage & Network": Server,
};

export function NodeReference({ initialNodeType, onSelectNode }: NodeReferenceProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedNodeType, setSelectedNodeType] = useState<string | null>(
    initialNodeType || null
  );
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    () => new Set(CATEGORY_ORDER)
  );

  const nodesByCategory = useMemo(() => getNodeDocumentationByCategory(), []);

  const filteredNodesByCategory = useMemo(() => {
    if (!searchQuery.trim()) {
      return nodesByCategory;
    }

    const query = searchQuery.toLowerCase();
    const filtered: Record<string, NodeDocEntry[]> = {};

    for (const [category, nodes] of Object.entries(nodesByCategory)) {
      const matchingNodes = nodes.filter(
        (node) =>
          node.name.toLowerCase().includes(query) ||
          node.summary.toLowerCase().includes(query) ||
          node.type.toLowerCase().includes(query)
      );
      if (matchingNodes.length > 0) {
        filtered[category] = matchingNodes;
      }
    }

    return filtered;
  }, [nodesByCategory, searchQuery]);

  const selectedDoc = selectedNodeType
    ? NODE_DOCUMENTATION[selectedNodeType]
    : null;

  const selectedNodeDef = selectedNodeType
    ? WORKFLOW_NODE_DEFINITIONS[selectedNodeType]
    : null;

  const toggleCategory = (category: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  const handleSelectNode = (nodeType: string) => {
    setSelectedNodeType(nodeType);
    onSelectNode?.(nodeType);
  };

  const totalNodes = Object.values(filteredNodesByCategory).flat().length;

  return (
    <div className="flex h-full">
      {/* Sidebar - Node List */}
      <div className="w-72 flex-shrink-0 border-r border-gray-800 flex flex-col bg-flow-bg">
        {/* Search */}
        <div className="p-4 border-b border-gray-800">
          <div className="relative">
            <Search
              size={16}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
            />
            <input
              type="search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search nodes..."
              className="w-full bg-gray-800/50 border border-gray-700 rounded-lg py-2 pl-10 pr-4 text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-flow-accent"
            />
          </div>
          <div className="mt-2 text-xs text-gray-500">
            {totalNodes} node{totalNodes !== 1 ? "s" : ""} available
          </div>
        </div>

        {/* Category List */}
        <div className="flex-1 overflow-y-auto">
          {CATEGORY_ORDER.map((category) => {
            const nodes = filteredNodesByCategory[category];
            if (!nodes || nodes.length === 0) return null;

            const isExpanded = expandedCategories.has(category);
            const CategoryIcon = CATEGORY_ICONS[category] || Database;

            return (
              <div key={category} className="border-b border-gray-800/50">
                <button
                  type="button"
                  onClick={() => toggleCategory(category)}
                  className="w-full flex items-center gap-2 px-4 py-3 text-left hover:bg-gray-800/30 transition-colors"
                >
                  <CategoryIcon size={16} className="text-flow-accent flex-shrink-0" />
                  <span className="text-sm font-medium text-gray-300 flex-1">
                    {category}
                  </span>
                  <span className="text-xs text-gray-500 mr-1">
                    {nodes.length}
                  </span>
                  {isExpanded ? (
                    <ChevronDown size={14} className="text-gray-500" />
                  ) : (
                    <ChevronRight size={14} className="text-gray-500" />
                  )}
                </button>

                {isExpanded && (
                  <div className="pb-2">
                    {nodes.map((node) => {
                      const nodeDef = WORKFLOW_NODE_DEFINITIONS[node.type];
                      const NodeIcon = nodeDef?.icon;
                      const isSelected = selectedNodeType === node.type;

                      return (
                        <button
                          key={node.type}
                          type="button"
                          onClick={() => handleSelectNode(node.type)}
                          className={`w-full flex items-start gap-3 px-4 py-2 text-left transition-colors ${
                            isSelected
                              ? "bg-flow-accent/20 border-l-2 border-flow-accent"
                              : "hover:bg-gray-800/30 border-l-2 border-transparent"
                          }`}
                        >
                          {NodeIcon && (
                            <NodeIcon
                              size={16}
                              className={`mt-0.5 flex-shrink-0 ${nodeDef?.color || "text-gray-400"}`}
                            />
                          )}
                          <div className="min-w-0 flex-1">
                            <div
                              className={`text-sm font-medium ${
                                isSelected ? "text-white" : "text-gray-300"
                              }`}
                            >
                              {node.name}
                            </div>
                            <div className="text-xs text-gray-500 truncate">
                              {node.summary}
                            </div>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}

          {Object.keys(filteredNodesByCategory).length === 0 && (
            <div className="p-4 text-center text-gray-500 text-sm">
              No nodes match "{searchQuery}"
            </div>
          )}
        </div>
      </div>

      {/* Main Content - Documentation */}
      <div className="flex-1 overflow-y-auto">
        {selectedDoc ? (
          <div className="p-6 max-w-4xl">
            {/* Header with icon */}
            <div className="flex items-center gap-3 mb-6 pb-4 border-b border-gray-800">
              {selectedNodeDef?.icon && (
                <div
                  className={`p-3 rounded-lg bg-gray-800/50 ${selectedNodeDef.color}`}
                >
                  <selectedNodeDef.icon size={24} />
                </div>
              )}
              <div>
                <h1 className="text-2xl font-bold text-white">
                  {selectedDoc.name}
                </h1>
                <div className="text-sm text-gray-400 mt-1">
                  {selectedDoc.category} • Type: <code className="text-amber-300">{selectedDoc.type}</code>
                </div>
              </div>
            </div>

            {/* Markdown content */}
            <MarkdownRenderer content={selectedDoc.content} />
          </div>
        ) : (
          <div className="flex items-center justify-center h-full text-gray-500">
            <div className="text-center">
              <Database size={48} className="mx-auto mb-4 opacity-50" />
              <p className="text-lg">Select a node to view its documentation</p>
              <p className="text-sm mt-2">
                Browse by category or use the search to find specific nodes
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
