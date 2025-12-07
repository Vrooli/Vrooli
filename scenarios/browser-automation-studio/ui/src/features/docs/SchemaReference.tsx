import { useState, useEffect } from "react";
import {
  FileJson,
  Copy,
  Check,
  ChevronRight,
  ChevronDown,
  ExternalLink,
} from "lucide-react";
import { MarkdownRenderer } from "./MarkdownRenderer";
import { WORKFLOW_SCHEMA_INTRO } from "./content/gettingStarted";

interface SchemaReferenceProps {
  apiBaseUrl?: string;
}

// Embedded schema fallback - this matches the Go validator schema structure
const EMBEDDED_SCHEMA = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  $id: "https://schemas.vrooli.com/browser-automation-studio/workflow.schema.json",
  title: "Vrooli Ascension Workflow",
  description: "Canonical schema for Vrooli Ascension workflow definitions",
  type: "object",
  required: ["nodes", "edges"],
  properties: {
    metadata: {
      type: "object",
      properties: {
        description: { type: "string" },
        requirement: { type: "string", pattern: "^[A-Za-z0-9-_:\\.]+$" },
        version: { type: ["number", "string"] },
        owner: { type: "string" },
      },
      additionalProperties: true,
    },
    settings: {
      type: "object",
      properties: {
        executionViewport: {
          type: "object",
          properties: {
            width: { type: "number", minimum: 200 },
            height: { type: "number", minimum: 200 },
            preset: { type: "string", enum: ["desktop", "mobile", "custom"] },
          },
          required: ["width", "height"],
        },
        entrySelector: { type: "string", minLength: 1 },
        entrySelectorTimeoutMs: { type: "number", minimum: 0 },
        entryTimeoutMs: { type: "number", minimum: 0 },
      },
    },
    nodes: {
      type: "array",
      minItems: 1,
      items: { $ref: "#/definitions/node" },
    },
    edges: {
      type: "array",
      items: { $ref: "#/definitions/edge" },
    },
  },
  definitions: {
    position: {
      type: "object",
      required: ["x", "y"],
      properties: {
        x: { type: "number" },
        y: { type: "number" },
      },
    },
    node: {
      type: "object",
      required: ["id", "type", "data"],
      properties: {
        id: { type: "string", minLength: 1 },
        type: { type: "string", minLength: 1 },
        position: {
          description: "Optional: omit or set to null for auto-layout workflows",
          anyOf: [{ $ref: "#/definitions/position" }, { type: "null" }],
        },
        data: { type: "object" },
        width: { type: "number" },
        height: { type: "number" },
      },
    },
    edge: {
      type: "object",
      required: ["id", "source", "target"],
      properties: {
        id: { type: "string", minLength: 1 },
        source: { type: "string", minLength: 1 },
        target: { type: "string", minLength: 1 },
        sourceHandle: { type: "string" },
        targetHandle: { type: "string" },
      },
    },
  },
};

interface SchemaPropertyProps {
  name: string;
  schema: Record<string, unknown>;
  required?: boolean;
  depth?: number;
}

function SchemaProperty({ name, schema, required = false, depth = 0 }: SchemaPropertyProps) {
  const [isExpanded, setIsExpanded] = useState(depth < 2);

  const type = schema.type as string | string[] | undefined;
  const description = schema.description as string | undefined;
  const properties = schema.properties as Record<string, unknown> | undefined;
  const items = schema.items as Record<string, unknown> | undefined;
  const requiredProps = schema.required as string[] | undefined;
  const hasChildren = properties || items;

  const typeDisplay = Array.isArray(type) ? type.join(" | ") : type;

  return (
    <div
      className={`${depth > 0 ? "ml-4 pl-4 border-l border-gray-800" : ""}`}
    >
      <div
        className={`flex items-start gap-2 py-1.5 ${hasChildren ? "cursor-pointer hover:bg-gray-800/30 -mx-2 px-2 rounded" : ""}`}
        onClick={() => hasChildren && setIsExpanded(!isExpanded)}
      >
        {hasChildren && (
          <span className="mt-1 text-gray-500">
            {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          </span>
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <code className="text-amber-300 text-sm">{name}</code>
            {typeDisplay && (
              <span className="text-xs text-purple-400 bg-purple-400/10 px-1.5 py-0.5 rounded">
                {typeDisplay}
              </span>
            )}
            {required && (
              <span className="text-xs text-rose-400 bg-rose-400/10 px-1.5 py-0.5 rounded">
                required
              </span>
            )}
          </div>
          {description && (
            <p className="text-xs text-gray-500 mt-0.5">{description}</p>
          )}
        </div>
      </div>

      {isExpanded && properties && (
        <div className="mt-1">
          {Object.entries(properties).map(([propName, propSchema]) => (
            <SchemaProperty
              key={propName}
              name={propName}
              schema={propSchema as Record<string, unknown>}
              required={requiredProps?.includes(propName)}
              depth={depth + 1}
            />
          ))}
        </div>
      )}

      {isExpanded && items && !properties && (
        <div className="mt-1 ml-4 pl-4 border-l border-gray-800">
          <div className="text-xs text-gray-500 mb-1">Array items:</div>
          {(items as Record<string, unknown>).$ref ? (
            <code className="text-xs text-cyan-400">
              {String((items as Record<string, unknown>).$ref)}
            </code>
          ) : (
            <SchemaProperty
              name="[item]"
              schema={items as Record<string, unknown>}
              depth={depth + 1}
            />
          )}
        </div>
      )}
    </div>
  );
}

export function SchemaReference({ apiBaseUrl }: SchemaReferenceProps) {
  const [schema, setSchema] = useState<Record<string, unknown>>(EMBEDDED_SCHEMA);
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState<"overview" | "schema" | "definitions">("overview");

  // Try to fetch live schema from API
  useEffect(() => {
    if (!apiBaseUrl) return;

    fetch(`${apiBaseUrl}/api/schema/workflow`)
      .then((res) => res.json())
      .then((data) => {
        if (data && typeof data === "object") {
          setSchema(data);
        }
      })
      .catch(() => {
        // Use embedded schema as fallback
      });
  }, [apiBaseUrl]);

  const copySchema = async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(schema, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy schema:", err);
    }
  };

  const properties = schema.properties as Record<string, unknown> | undefined;
  const definitions = schema.definitions as Record<string, unknown> | undefined;
  const requiredProps = schema.required as string[] | undefined;

  return (
    <div className="h-full flex flex-col">
      {/* Tab Navigation */}
      <div className="flex border-b border-gray-800 px-4">
        {(
          [
            { id: "overview", label: "Overview" },
            { id: "schema", label: "Schema Explorer" },
            { id: "definitions", label: "Type Definitions" },
          ] as const
        ).map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab.id
                ? "border-flow-accent text-white"
                : "border-transparent text-gray-400 hover:text-gray-300"
            }`}
          >
            {tab.label}
          </button>
        ))}
        <div className="ml-auto flex items-center gap-2 py-2">
          <button
            type="button"
            onClick={copySchema}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          >
            {copied ? <Check size={14} /> : <Copy size={14} />}
            {copied ? "Copied!" : "Copy JSON"}
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === "overview" && (
          <div className="max-w-4xl">
            <MarkdownRenderer content={WORKFLOW_SCHEMA_INTRO} />
          </div>
        )}

        {activeTab === "schema" && (
          <div className="max-w-4xl">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 bg-amber-500/20 rounded-lg">
                <FileJson size={24} className="text-amber-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">
                  {schema.title as string || "Workflow Schema"}
                </h2>
                <p className="text-sm text-gray-400">
                  {schema.description as string}
                </p>
              </div>
            </div>

            <div className="mb-4 p-3 bg-gray-800/50 rounded-lg border border-gray-700">
              <div className="flex items-center gap-2 text-xs text-gray-400">
                <ExternalLink size={12} />
                <span>Schema ID:</span>
                <code className="text-cyan-400">{schema.$id as string}</code>
              </div>
            </div>

            <h3 className="text-lg font-semibold text-white mb-3">Properties</h3>
            <div className="bg-gray-900/50 rounded-lg border border-gray-800 p-4">
              {properties &&
                Object.entries(properties).map(([name, propSchema]) => (
                  <SchemaProperty
                    key={name}
                    name={name}
                    schema={propSchema as Record<string, unknown>}
                    required={requiredProps?.includes(name)}
                  />
                ))}
            </div>
          </div>
        )}

        {activeTab === "definitions" && (
          <div className="max-w-4xl">
            <h2 className="text-xl font-bold text-white mb-4">Type Definitions</h2>
            <p className="text-gray-400 mb-6">
              These are the reusable type definitions referenced throughout the schema.
            </p>

            {definitions && (
              <div className="space-y-6">
                {Object.entries(definitions).map(([name, defSchema]) => {
                  const def = defSchema as Record<string, unknown>;
                  const defProperties = def.properties as Record<string, unknown> | undefined;
                  const defRequired = def.required as string[] | undefined;

                  return (
                    <div
                      key={name}
                      className="bg-gray-900/50 rounded-lg border border-gray-800 p-4"
                    >
                      <h3 className="text-lg font-semibold text-white mb-2 flex items-center gap-2">
                        <code className="text-cyan-400">#{name}</code>
                        {typeof def.type === "string" && (
                          <span className="text-xs text-purple-400 bg-purple-400/10 px-1.5 py-0.5 rounded">
                            {def.type}
                          </span>
                        )}
                      </h3>
                      {typeof def.description === "string" && (
                        <p className="text-sm text-gray-400 mb-3">
                          {def.description}
                        </p>
                      )}
                      {defProperties && (
                        <div className="mt-3 pt-3 border-t border-gray-800">
                          {Object.entries(defProperties).map(([propName, propSchema]) => (
                            <SchemaProperty
                              key={propName}
                              name={propName}
                              schema={propSchema as Record<string, unknown>}
                              required={defRequired?.includes(propName)}
                            />
                          ))}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
