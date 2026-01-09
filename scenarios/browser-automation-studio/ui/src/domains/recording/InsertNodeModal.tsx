/**
 * InsertNodeModal - Modal for inserting new action steps into the recording timeline.
 *
 * This modal reuses the node category structure from the NodePalette to provide
 * a consistent experience for selecting and configuring nodes.
 */

import { useState, useCallback, useMemo, useId } from "react";
import {
  X,
  ChevronLeft,
  ChevronDown,
  Search,
  Plus,
} from "lucide-react";
import { ResponsiveDialog } from "@shared/layout";
import {
  NODE_CATEGORIES,
  WORKFLOW_NODE_DEFINITIONS,
  type WorkflowNodeDefinition,
  type NodeCategory,
} from "@constants/nodeCategories";

type InsertNodeStage = "browse" | "configure";

interface InsertNodeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onInsert: (action: InsertedAction) => void;
}

// The action structure that will be inserted into the timeline
export interface InsertedAction {
  type: string;
  params: Record<string, unknown>;
  label?: string;
}

// Node configurations for the configure stage
interface NodeConfigField {
  key: string;
  label: string;
  type: "text" | "textarea" | "select" | "number" | "checkbox";
  placeholder?: string;
  required?: boolean;
  options?: { value: string; label: string }[];
  defaultValue?: string | number | boolean;
}

// Configuration schemas for different node types
const NODE_CONFIG_SCHEMAS: Record<string, NodeConfigField[]> = {
  assert: [
    {
      key: "selector",
      label: "CSS Selector",
      type: "text",
      placeholder: "#element, .class, [data-testid='value']",
      required: true,
    },
    {
      key: "mode",
      label: "Assertion Mode",
      type: "select",
      required: true,
      options: [
        { value: "exists", label: "Element Exists" },
        { value: "not_exists", label: "Element Does Not Exist" },
        { value: "visible", label: "Element Visible" },
        { value: "text_equals", label: "Text Equals" },
        { value: "text_contains", label: "Text Contains" },
        { value: "attribute_equals", label: "Attribute Equals" },
      ],
      defaultValue: "exists",
    },
    {
      key: "expected",
      label: "Expected Value",
      type: "text",
      placeholder: "Expected text or attribute value",
    },
    {
      key: "attribute",
      label: "Attribute Name",
      type: "text",
      placeholder: "e.g., href, data-state",
    },
    {
      key: "caseSensitive",
      label: "Case Sensitive",
      type: "checkbox",
      defaultValue: false,
    },
  ],
  wait: [
    {
      key: "waitType",
      label: "Wait Type",
      type: "select",
      required: true,
      options: [
        { value: "selector", label: "Wait for Element" },
        { value: "time", label: "Wait for Duration" },
        { value: "navigation", label: "Wait for Navigation" },
        { value: "networkIdle", label: "Wait for Network Idle" },
      ],
      defaultValue: "selector",
    },
    {
      key: "selector",
      label: "CSS Selector",
      type: "text",
      placeholder: "#element, .class, [data-testid='value']",
    },
    {
      key: "duration",
      label: "Duration (ms)",
      type: "number",
      placeholder: "1000",
      defaultValue: 1000,
    },
    {
      key: "state",
      label: "Element State",
      type: "select",
      options: [
        { value: "visible", label: "Visible" },
        { value: "hidden", label: "Hidden" },
        { value: "attached", label: "Attached to DOM" },
        { value: "detached", label: "Detached from DOM" },
      ],
      defaultValue: "visible",
    },
  ],
  screenshot: [
    {
      key: "name",
      label: "Screenshot Name",
      type: "text",
      placeholder: "screenshot-name",
    },
    {
      key: "fullPage",
      label: "Full Page Screenshot",
      type: "checkbox",
      defaultValue: false,
    },
    {
      key: "selector",
      label: "Element Selector (optional)",
      type: "text",
      placeholder: "Capture specific element",
    },
  ],
  extract: [
    {
      key: "selector",
      label: "CSS Selector",
      type: "text",
      placeholder: "#element, .class",
      required: true,
    },
    {
      key: "extractType",
      label: "Extract Type",
      type: "select",
      required: true,
      options: [
        { value: "text", label: "Text Content" },
        { value: "innerHTML", label: "Inner HTML" },
        { value: "attribute", label: "Attribute Value" },
        { value: "value", label: "Input Value" },
      ],
      defaultValue: "text",
    },
    {
      key: "attribute",
      label: "Attribute Name",
      type: "text",
      placeholder: "e.g., href, src",
    },
    {
      key: "variableName",
      label: "Save to Variable",
      type: "text",
      placeholder: "myVariable",
    },
  ],
  setVariable: [
    {
      key: "name",
      label: "Variable Name",
      type: "text",
      placeholder: "myVariable",
      required: true,
    },
    {
      key: "value",
      label: "Value",
      type: "textarea",
      placeholder: "Variable value (can use {{otherVar}})",
      required: true,
    },
  ],
  evaluate: [
    {
      key: "script",
      label: "JavaScript Code",
      type: "textarea",
      placeholder: "return document.title;",
      required: true,
    },
    {
      key: "variableName",
      label: "Save Result to Variable",
      type: "text",
      placeholder: "result",
    },
  ],
};

// Filter to only show the most useful nodes for manual insertion
const INSERTABLE_NODE_TYPES = [
  "assert",
  "wait",
  "screenshot",
  "extract",
  "setVariable",
  "evaluate",
  "conditional",
  "loop",
];

export function InsertNodeModal({
  isOpen,
  onClose,
  onInsert,
}: InsertNodeModalProps) {
  const titleId = useId();
  const [stage, setStage] = useState<InsertNodeStage>("browse");
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedNode, setSelectedNode] = useState<WorkflowNodeDefinition | null>(null);
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    () => new Set(["assertions", "data"])
  );

  // Filter nodes to only show insertable ones
  const filteredCategories = useMemo(() => {
    const query = searchTerm.toLowerCase().trim();

    return NODE_CATEGORIES.map((category) => {
      const nodes = category.nodes
        .filter((nodeType) => INSERTABLE_NODE_TYPES.includes(nodeType))
        .map((nodeType) => WORKFLOW_NODE_DEFINITIONS[nodeType])
        .filter((node): node is WorkflowNodeDefinition => {
          if (!node) return false;
          if (!query) return true;
          return (
            node.label.toLowerCase().includes(query) ||
            node.description.toLowerCase().includes(query) ||
            (node.keywords?.some((k) => k.toLowerCase().includes(query)) ?? false)
          );
        });

      if (nodes.length === 0) return null;
      return { ...category, nodes };
    }).filter(Boolean) as Array<Omit<NodeCategory, 'nodes'> & { nodes: WorkflowNodeDefinition[] }>;
  }, [searchTerm]);

  const handleNodeSelect = useCallback((node: WorkflowNodeDefinition) => {
    setSelectedNode(node);
    // Initialize config with defaults
    const schema = NODE_CONFIG_SCHEMAS[node.type] || [];
    const defaults: Record<string, unknown> = {};
    for (const field of schema) {
      if (field.defaultValue !== undefined) {
        defaults[field.key] = field.defaultValue;
      }
    }
    setConfigValues(defaults);
    setStage("configure");
  }, []);

  const handleBack = useCallback(() => {
    setStage("browse");
    setSelectedNode(null);
    setConfigValues({});
  }, []);

  const handleConfigChange = useCallback((key: string, value: unknown) => {
    setConfigValues((prev) => ({ ...prev, [key]: value }));
  }, []);

  const handleInsert = useCallback(() => {
    if (!selectedNode) return;

    const action: InsertedAction = {
      type: selectedNode.type,
      params: { ...configValues },
      label: selectedNode.label,
    };

    onInsert(action);
    handleClose();
  }, [selectedNode, configValues, onInsert]);

  const handleClose = useCallback(() => {
    setStage("browse");
    setSelectedNode(null);
    setConfigValues({});
    setSearchTerm("");
    onClose();
  }, [onClose]);

  const toggleCategory = useCallback((categoryId: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(categoryId)) {
        next.delete(categoryId);
      } else {
        next.add(categoryId);
      }
      return next;
    });
  }, []);

  // Check if required fields are filled
  const isConfigValid = useMemo(() => {
    if (!selectedNode) return false;
    const schema = NODE_CONFIG_SCHEMAS[selectedNode.type] || [];
    return schema
      .filter((field) => field.required)
      .every((field) => {
        const value = configValues[field.key];
        return value !== undefined && value !== "";
      });
  }, [selectedNode, configValues]);

  const configSchema = selectedNode ? NODE_CONFIG_SCHEMAS[selectedNode.type] || [] : [];

  if (!isOpen) return null;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={handleClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden max-h-[80vh] flex flex-col"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700">
        <div className="flex items-center gap-2">
          {stage === "configure" && (
            <button
              onClick={handleBack}
              className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
              aria-label="Back to browse"
            >
              <ChevronLeft size={18} />
            </button>
          )}
          <div className="flex items-center gap-2">
            {selectedNode ? (
              <>
                <selectedNode.icon size={18} className={selectedNode.color} />
                <h2 id={titleId} className="text-lg font-semibold text-white">
                  Configure {selectedNode.label}
                </h2>
              </>
            ) : (
              <>
                <Plus size={18} className="text-blue-400" />
                <h2 id={titleId} className="text-lg font-semibold text-white">
                  Insert Step
                </h2>
              </>
            )}
          </div>
        </div>
        <button
          onClick={handleClose}
          className="p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
          aria-label="Close"
        >
          <X size={18} />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {stage === "browse" ? (
          <>
            {/* Search */}
            <div className="relative mb-4">
              <Search
                size={14}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
              />
              <input
                type="search"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search nodes..."
                className="w-full bg-gray-800/50 border border-gray-700 rounded-lg py-2 pl-9 pr-3 text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* Categories */}
            <div className="space-y-3">
              {filteredCategories.map((category) => {
                const isExpanded = searchTerm ? true : expandedCategories.has(category.id);
                const CategoryIcon = category.icon;

                return (
                  <div
                    key={category.id}
                    className="border border-gray-800 rounded-lg overflow-hidden"
                  >
                    <button
                      type="button"
                      onClick={() => toggleCategory(category.id)}
                      className="w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-gray-800/50 transition-colors"
                      disabled={Boolean(searchTerm)}
                    >
                      <CategoryIcon size={16} className="text-blue-400" />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-white">
                          {category.label}
                        </div>
                        <div className="text-xs text-gray-500">
                          {category.description}
                        </div>
                      </div>
                      <ChevronDown
                        size={14}
                        className={`text-gray-500 transition-transform ${
                          isExpanded ? "rotate-0" : "-rotate-90"
                        }`}
                      />
                    </button>
                    {isExpanded && (
                      <div className="p-2 space-y-2 bg-gray-800/30">
                        {category.nodes.map((node) => {
                          const Icon = node.icon;
                          return (
                            <button
                              key={node.type}
                              onClick={() => handleNodeSelect(node)}
                              className="w-full flex items-center gap-3 p-3 bg-gray-800/50 border border-gray-700 rounded-lg hover:border-blue-500/50 hover:bg-gray-800 transition-all text-left group"
                            >
                              <div className={`${node.color} group-hover:scale-110 transition-transform`}>
                                <Icon size={18} />
                              </div>
                              <div className="flex-1">
                                <div className="font-medium text-sm text-white">
                                  {node.label}
                                </div>
                                <div className="text-xs text-gray-500">
                                  {node.description}
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
            </div>

            {filteredCategories.length === 0 && searchTerm && (
              <div className="text-center text-gray-500 py-8">
                No nodes match "{searchTerm}"
              </div>
            )}
          </>
        ) : (
          /* Configure Stage */
          <div className="space-y-4">
            {configSchema.length === 0 ? (
              <div className="text-center text-gray-400 py-4">
                <p>This node has no configurable options.</p>
                <p className="text-sm text-gray-500 mt-2">
                  Click "Insert Step" to add it to the timeline.
                </p>
              </div>
            ) : (
              configSchema.map((field) => {
                // Hide fields based on other field values
                if (field.key === "selector" && selectedNode?.type === "wait") {
                  if (configValues.waitType !== "selector") return null;
                }
                if (field.key === "duration" && selectedNode?.type === "wait") {
                  if (configValues.waitType !== "time") return null;
                }
                if (field.key === "state" && selectedNode?.type === "wait") {
                  if (configValues.waitType !== "selector") return null;
                }
                if (field.key === "expected" && selectedNode?.type === "assert") {
                  const mode = configValues.mode as string;
                  if (!["text_equals", "text_contains", "attribute_equals"].includes(mode)) return null;
                }
                if (field.key === "attribute") {
                  if (selectedNode?.type === "assert" && configValues.mode !== "attribute_equals") return null;
                  if (selectedNode?.type === "extract" && configValues.extractType !== "attribute") return null;
                }
                if (field.key === "caseSensitive" && selectedNode?.type === "assert") {
                  const mode = configValues.mode as string;
                  if (!["text_equals", "text_contains", "attribute_equals"].includes(mode)) return null;
                }

                return (
                  <div key={field.key}>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      {field.label}
                      {field.required && <span className="text-red-400 ml-1">*</span>}
                    </label>
                    {field.type === "select" ? (
                      <select
                        value={(configValues[field.key] as string) ?? ""}
                        onChange={(e) => handleConfigChange(field.key, e.target.value)}
                        className="w-full px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-blue-500"
                      >
                        {field.options?.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </select>
                    ) : field.type === "textarea" ? (
                      <textarea
                        value={(configValues[field.key] as string) ?? ""}
                        onChange={(e) => handleConfigChange(field.key, e.target.value)}
                        placeholder={field.placeholder}
                        rows={3}
                        className="w-full px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white placeholder:text-gray-500 focus:outline-none focus:border-blue-500 resize-none font-mono text-sm"
                      />
                    ) : field.type === "checkbox" ? (
                      <label className="flex items-center gap-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={Boolean(configValues[field.key])}
                          onChange={(e) => handleConfigChange(field.key, e.target.checked)}
                          className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-500 focus:ring-blue-500"
                        />
                        <span className="text-sm text-gray-400">Enabled</span>
                      </label>
                    ) : field.type === "number" ? (
                      <input
                        type="number"
                        value={(configValues[field.key] as number) ?? ""}
                        onChange={(e) => handleConfigChange(field.key, parseInt(e.target.value, 10) || 0)}
                        placeholder={field.placeholder}
                        className="w-full px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white placeholder:text-gray-500 focus:outline-none focus:border-blue-500"
                      />
                    ) : (
                      <input
                        type="text"
                        value={(configValues[field.key] as string) ?? ""}
                        onChange={(e) => handleConfigChange(field.key, e.target.value)}
                        placeholder={field.placeholder}
                        className="w-full px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white placeholder:text-gray-500 focus:outline-none focus:border-blue-500"
                      />
                    )}
                  </div>
                );
              })
            )}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="flex items-center justify-end gap-3 px-4 py-3 border-t border-gray-700 bg-gray-800/50">
        <button
          onClick={handleClose}
          className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors"
        >
          Cancel
        </button>
        {stage === "configure" && (
          <button
            onClick={handleInsert}
            disabled={!isConfigValid}
            className="px-4 py-2 bg-blue-500 text-white text-sm font-medium rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
          >
            <Plus size={14} />
            Insert Step
          </button>
        )}
      </div>
    </ResponsiveDialog>
  );
}

export default InsertNodeModal;
