/* ========================================================================== 
 *  generate-tool-schemas.ts                                                  
 * -------------------------------------------------------------------------- 
 *  Generates JSON-Schema definitions for every tool that our agents may use. 
 *                                                                            
 *  There are **three** broad buckets of schemas we generate:     
 * 
 *  1. **DefineTool schemas** – Schemas explaining how exactly to find, add, 
 *     or update a resource of a specific type. You can only get these by calling 
 *     the DefineTool becuase otherwise they would unnecessarily bloat the context window.
 *                                                                            
 *  2. **MCP tool schemas** – Schemas for calling top-level MCP tools, such as 
 *     DefineTool, ResourceManage, SpawnSwarm, UpdateSwarmSharedState, etc.
 *                                                                            
 *  3. **AI-service-native tools** – e.g. the built-in `file_search` and      
 *     `code_interpreter` tools exposed by the OpenAI API. The relevant      
 *     shapes are derived from the *official* service SDK types so that our   
 *     schemas always stay in-lock-step with upstream changes.                                                               *
 *                                                                            
 *  The output directory layout therefore looks like:                         
 *                                                                            
 *  └─ src                                                                    
 *     └─ services                                                            
 *        └─ schemas              
 *           │  // Built-in tools      
 *           ├─ DefineTool       
 *           │  ├─ Bot
 *           │  │  ├─ add_attributes.json
 *           │  │  ├─ update_attributes.json
 *           │  │  └─ find_attributes.json
 *           │  ├─ Team
 *           │  │  ├─ add_attributes.json
 *           │  │  //...
 *           │  └─ schema.json
 *           ├─ SendMessage
 *           │  └─ schema.json
 *           ├─ ResourceManage
 *           │  └─ schema.json
 *           ├─ RunRoutine
 *           │  └─ schema.json
 *           ├─ SpawnSwarm
 *           │  └─ schema.json
 *           │  // Swarm-level tools      
 *           ├─ UpdateSwarmSharedState   
 *           │  └─ schema.json  
 *           ├─ EndSwarm     
 *           │  └─ schema.json
 *           │  // Routine-level tools      
 *           ├─ StartRoutine           
 *           │  └─ schema.json
 *           ├─ StopRoutine
 *           │  └─ schema.json
 *           │  // AI service tools       
 *           └─ services          
 *              ├─ openai    
 *              │  ├─ code_interpreter.json
 *              │  ├─ image_generation.json
 *              │  ├─ mcp.json
 *              │  ├─ file_search.json
 *              │  ├─ web_search_preview.json
 *              │  └─ computer_use_preview.json
 *              ├─ anthropic              
 *              └─ ...                        
 *                                                   
 *  Each directory contains plain JSON-Schema files ready for runtime         
 *  validation **and** for feeding straight back into LLMs via                
 *  `tool_choice`.  Trimming Titles & boilerplate keeps payloads small so we  
 *  can send many of them in the same prompt.                                 
 * ========================================================================== */
import { McpSwarmToolName, McpToolName } from "@vrooli/shared";
import * as fs from "fs";
import * as path from "path";
import ts from "typescript"; // Import the 'ts' namespace for enums
import * as TJS from "typescript-json-schema";
import { fileURLToPath } from "url";
import { McpRoutineToolName } from "../../services/types/tools.js";
// We need the Responses namespace to access the built-in Tool types
// eslint-disable-next-line @typescript-eslint/no-unused-vars

/* ──────────────────────────── Paths & Constants ─────────────────────────── */

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Monorepo roots -----------------------------------------------------------------
const packageServerRoot = path.resolve(__dirname, "../../..");         // packages/server/
const monorepoRoot = path.resolve(packageServerRoot, "..");       // repo root

// Input locations --------------------------------------------------------------------
const IN_TYPES_BASE = path.resolve(packageServerRoot, "src/services/types");
const IN_TYPES_RESOURCES = path.resolve(IN_TYPES_BASE, "resources.ts");
const IN_TYPES_SERVICES = path.resolve(IN_TYPES_BASE, "services.ts");
const IN_TYPES_TOOLS = path.resolve(IN_TYPES_BASE, "tools.ts");

// 2. Output roots -------------------------------------------------------------------
const OUT_SCHEMAS_BASE = path.resolve(packageServerRoot, "src/services/schemas");
const OUT_DEFINE_TOOL_BASE = path.resolve(OUT_SCHEMAS_BASE, "DefineTool");
const OUT_SERVICES_BASE = path.resolve(OUT_SCHEMAS_BASE, "services");

// Track how many schema generation failures occur
let failureCount = 0;

/* ───────────────────────────── Utility helpers ──────────────────────────── */

/**
 * Recursively strips `title` keys from a JSON-Schema for smaller payloads.
 * @param node   Any JSON-Schema node.
 * @param isRoot Whether the current node is the root (keep root title for clarity).
 */
function stripTitles(node: unknown, isRoot = false): void {
    if (!node || typeof node !== "object") return;

    if (!isRoot && "title" in (node as Record<string, unknown>)) {
        delete (node as Record<string, unknown>).title;
    }
    if (Array.isArray(node)) {
        node.forEach(n => stripTitles(n, false));
    } else {
        Object.values(node).forEach(n => stripTitles(n, false));
    }
}

/**
 * Removes noise that TypeScript-JSON-Schema tends to emit but which is not
 * valuable for LLM consumption (`defaultProperties: []`, `additionalProperties:false`, …).
 */
function stripBoilerplate(node: unknown): void {
    if (Array.isArray(node)) {
        node.forEach(stripBoilerplate);
        return;
    }

    if (node && typeof node === "object") {
        const o = node as Record<string, unknown>;

        if (
            "defaultProperties" in o &&
            Array.isArray(o.defaultProperties) &&
            o.defaultProperties.length === 0
        ) delete o.defaultProperties;

        if (o.additionalProperties === false) delete o.additionalProperties;

        Object.values(o).forEach(stripBoilerplate);
    }
}

/* ──────────────────────────── TS-JSON-Schema cfg ────────────────────────── */

const baseSettings: TJS.PartialArgs = {
    required: true,
    titles: true,
    defaultProps: true,
    noExtraProps: true,
    ref: false,  // keep schemas flat – easier for agents to reason about
    ignoreErrors: true,
    validationKeywords: [
        "format",
        "pattern",
        "minLength",
        "maxLength",
        "minimum",
        "maximum",
        "enum",
        "default",
        "items",
        "properties",
    ],
};

const compilerOptions: TJS.CompilerOptions = {
    /* — Standard project settings — */
    allowJs: true,
    target: ts.ScriptTarget.ESNext,
    module: ts.ModuleKind.ESNext,
    moduleResolution: ts.ModuleResolutionKind.NodeNext,
    strictNullChecks: true,
    allowSyntheticDefaultImports: true,
    esModuleInterop: true,
    resolveJsonModule: true,
    skipLibCheck: true, // IMPORTANT: To avoid d.ts conflicts
    downlevelIteration: true,
    jsx: ts.JsxEmit.ReactJSX,
    noEmit: true,
    noEmitOnError: true,
    baseUrl: ".", // Relative to monorepoRoot (which will be TJS.getProgramFromFiles basePath)
    rootDir: ".", // All source files are under monorepoRoot
    lib: [
        "esnext", // Ensure esnext for modern features
        "dom",
        "dom.iterable",
    ],
    paths: {
        "@vrooli/shared": ["packages/shared/src"],      // Path from monorepoRoot
        "@vrooli/shared/*": ["packages/shared/src/*"],   // Path from monorepoRoot
    },
    // TypeRoots relative to monorepoRoot, derived from shared/tsconfig.json
    typeRoots: [
        "./node_modules/@types",                 // Project root node_modules/@types
        "./packages/shared/node_modules/@types", // If shared has its own node_modules/@types
        "./packages/shared/src/types",            // Custom types in shared package
    ],
    experimentalDecorators: true,
    emitDecoratorMetadata: true,
    forceConsistentCasingInFileNames: true,
    noFallthroughCasesInSwitch: true,
    noImplicitAny: false, // Matching typical project settings
    noImplicitThis: true, // Matching typical project settings
    // Ensure these are false for TJS compatibility/stability
    isolatedModules: false,
    declaration: false,
    incremental: false, // TJS might not need/want this
};

/* ─────────────────────── Map types to outputs ──────────────────────── */

type TypeToSchema = {
    /** The name of the file to write the schema to */
    fileName: string;
    /** The subdirectory to write the schema to, if any */
    subdir?: string;
    /** The name of the TypeScript type to generate a schema for */
    typeName: string;
    /** 
     * Whether to use references in the schema.
     * This decreases schema size, at the cost of making the schema less readable.
     * 
     * @default false
     */
    useRef?: boolean;
    /** Optional override for the root JSON-Schema title */
    name?: string;
    /** Optional top-level annotations object to add to the schema */
    annotations?: Record<string, unknown>;
    /** Optional hook to mutate or reshape the generated schema object before writing */
    transform?: (schema: Record<string, unknown>) => Record<string, unknown>;
};

/**
 * List of MCP variants & operations for DefineTool results.
 * 
 * "find" results should be used as the "filter" object when using the "find" op with ResourceManage.
 * "add" results should be used as the "attributes" object when using the "add" op with ResourceManage.
 * "update" results should be used as the "attributes" object when using the "update" op with ResourceManage.
 */
const RESOURCE_SCHEMAS: Array<TypeToSchema> = [
    // RoutineMultiStep
    { subdir: "RoutineMultiStep", fileName: "add_attributes", typeName: "RoutineMultiStepAddAttributes", useRef: true },
    { subdir: "RoutineMultiStep", fileName: "update_attributes", typeName: "RoutineMultiStepUpdateAttributes", useRef: true },
    { subdir: "RoutineMultiStep", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineInternalAction
    { subdir: "RoutineInternalAction", fileName: "add_attributes", typeName: "RoutineInternalActionAddAttributes", useRef: true },
    { subdir: "RoutineInternalAction", fileName: "update_attributes", typeName: "RoutineInternalActionUpdateAttributes", useRef: true },
    { subdir: "RoutineInternalAction", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineApi
    { subdir: "RoutineApi", fileName: "add_attributes", typeName: "RoutineApiAddAttributes", useRef: true },
    { subdir: "RoutineApi", fileName: "update_attributes", typeName: "RoutineApiUpdateAttributes", useRef: true },
    { subdir: "RoutineApi", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineCode
    { subdir: "RoutineCode", fileName: "add_attributes", typeName: "RoutineCodeAddAttributes", useRef: true },
    { subdir: "RoutineCode", fileName: "update_attributes", typeName: "RoutineCodeUpdateAttributes", useRef: true },
    { subdir: "RoutineCode", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineData
    { subdir: "RoutineData", fileName: "add_attributes", typeName: "RoutineDataAddAttributes", useRef: true },
    { subdir: "RoutineData", fileName: "update_attributes", typeName: "RoutineDataUpdateAttributes", useRef: true },
    { subdir: "RoutineData", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineGenerate
    { subdir: "RoutineGenerate", fileName: "add_attributes", typeName: "RoutineGenerateAddAttributes", useRef: true },
    { subdir: "RoutineGenerate", fileName: "update_attributes", typeName: "RoutineGenerateUpdateAttributes", useRef: true },
    { subdir: "RoutineGenerate", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineInformational
    { subdir: "RoutineInformational", fileName: "add_attributes", typeName: "RoutineInformationalAddAttributes", useRef: true },
    { subdir: "RoutineInformational", fileName: "update_attributes", typeName: "RoutineInformationalUpdateAttributes", useRef: true },
    { subdir: "RoutineInformational", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineSmartContract
    { subdir: "RoutineSmartContract", fileName: "add_attributes", typeName: "RoutineSmartContractAddAttributes", useRef: true },
    { subdir: "RoutineSmartContract", fileName: "update_attributes", typeName: "RoutineSmartContractUpdateAttributes", useRef: true },
    { subdir: "RoutineSmartContract", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // RoutineWeb
    { subdir: "RoutineWeb", fileName: "add_attributes", typeName: "RoutineWebAddAttributes", useRef: true },
    { subdir: "RoutineWeb", fileName: "update_attributes", typeName: "RoutineWebUpdateAttributes", useRef: true },
    { subdir: "RoutineWeb", fileName: "find_filters", typeName: "ResourceFindFilters", useRef: true },
    // Bot
    { subdir: "Bot", fileName: "add_attributes", typeName: "BotAddAttributes" },
    { subdir: "Bot", fileName: "update_attributes", typeName: "BotUpdateAttributes" },
    { subdir: "Bot", fileName: "find_filters", typeName: "BotFindFilters" },
    // Team
    { subdir: "Team", fileName: "add_attributes", typeName: "TeamAddAttributes" },
    { subdir: "Team", fileName: "update_attributes", typeName: "TeamUpdateAttributes" },
    { subdir: "Team", fileName: "find_filters", typeName: "TeamFindFilters" },
    // StandardPrompt
    { subdir: "StandardPrompt", fileName: "add_attributes", typeName: "StandardPromptAddAttributes", useRef: true },
    { subdir: "StandardPrompt", fileName: "update_attributes", typeName: "StandardPromptUpdateAttributes", useRef: true },
    { subdir: "StandardPrompt", fileName: "find_filters", typeName: "StandardPromptFindFilters", useRef: true },
    // StandardDataStructure
    { subdir: "StandardDataStructure", fileName: "add_attributes", typeName: "StandardDataStructureAddAttributes", useRef: true },
    { subdir: "StandardDataStructure", fileName: "update_attributes", typeName: "StandardDataStructureUpdateAttributes", useRef: true },
    { subdir: "StandardDataStructure", fileName: "find_filters", typeName: "StandardDataStructureFindFilters", useRef: true },
    // Project
    { subdir: "Project", fileName: "add_attributes", typeName: "ProjectAddAttributes" },
    { subdir: "Project", fileName: "update_attributes", typeName: "ProjectUpdateAttributes" },
    { subdir: "Project", fileName: "find_filters", typeName: "ProjectFindFilters" },
    // Note
    { subdir: "Note", fileName: "add_attributes", typeName: "NoteAddAttributes" },
    { subdir: "Note", fileName: "update_attributes", typeName: "NoteUpdateAttributes" },
    { subdir: "Note", fileName: "find_filters", typeName: "NoteFindFilters" },
    // ExternalData
    { subdir: "ExternalData", fileName: "add_attributes", typeName: "ExternalDataAddAttributes" },
    { subdir: "ExternalData", fileName: "update_attributes", typeName: "ExternalDataUpdateAttributes" },
    { subdir: "ExternalData", fileName: "find_filters", typeName: "ExternalDataFindFilters" },
];

// Add shared helper to properly format MCP tool schemas
function wrapInputSchema(schema: Record<string, unknown>): Record<string, unknown> {
    const { anyOf, oneOf, type, properties, required, $schema, ...rest } = schema;
    if (anyOf) {
        return { ...rest, inputSchema: { anyOf, additionalProperties: false } };
    }
    if (oneOf) {
        return { ...rest, inputSchema: { oneOf, additionalProperties: false } };
    }
    return { ...rest, inputSchema: { type, properties, required, additionalProperties: false } };
}

// List of top-level MCP tool schemas to generate
const MCP_TOOL_SCHEMAS: TypeToSchema[] = [
    // Built-in tools
    {
        subdir: "DefineTool",
        fileName: "schema",
        typeName: "DefineToolParams",
        name: McpToolName.DefineTool,
        transform: wrapInputSchema,
        annotations: {
            title: "Define Tool Parameters",
            readOnlyHint: true, // Does not modify state
            openWorldHint: false, // Does not interact with the real world
        },
    },
    {
        subdir: "SendMessage",
        fileName: "schema",
        typeName: "SendMessageParams",
        name: McpToolName.SendMessage,
        transform: wrapInputSchema,
        annotations: {
            title: "Send Message",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    },
    {
        subdir: "ResourceManage",
        fileName: "schema",
        typeName: "ResourceManageParams",
        name: McpToolName.ResourceManage,
        transform: wrapInputSchema,
        annotations: {
            title: "Resource Manage",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    },
    {
        subdir: "RunRoutine",
        fileName: "schema",
        typeName: "RunRoutineParams",
        name: McpToolName.RunRoutine,
        transform: wrapInputSchema,
        annotations: {
            title: "Run Routine",
            readOnlyHint: false, // Modifies state
            openWorldHint: true, // May interact with the real world
        },
    },
    {
        subdir: "SpawnSwarm",
        fileName: "schema",
        typeName: "SpawnSwarmParams",
        name: McpToolName.SpawnSwarm,
        transform: wrapInputSchema,
        annotations: {
            title: "Spawn Swarm",
            readOnlyHint: false, // Modifies state
            openWorldHint: true, // Interacts with the real world
        },
    },
    // Swarm-level tools
    {
        subdir: "UpdateSwarmSharedState",
        fileName: "schema",
        typeName: "UpdateSwarmSharedStateParams",
        name: McpSwarmToolName.UpdateSwarmSharedState,
        transform: wrapInputSchema,
        annotations: {
            title: "Update Swarm Shared State",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    },
    {
        subdir: "EndSwarm",
        fileName: "schema",
        typeName: "EndSwarmParams",
        name: McpSwarmToolName.EndSwarm,
        transform: wrapInputSchema,
        annotations: {
            title: "End Swarm",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    },
    // Routine-level tools
    {
        subdir: "StartRoutine",
        fileName: "schema",
        typeName: "StartRoutineParams",
        name: McpRoutineToolName.StartRoutine,
        transform: wrapInputSchema,
        annotations: {
            title: "Start Routine",
            readOnlyHint: false, // Modifies state
            openWorldHint: true, // Interacts with the real world
        },
    },
    {
        subdir: "StopRoutine",
        fileName: "schema",
        typeName: "StopRoutineParams",
        name: McpRoutineToolName.StopRoutine,
        transform: wrapInputSchema,
        annotations: {
            title: "Stop Routine",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    },
];

// Define the desired order of top-level fields in the output
const FIELD_ORDER = [
    '$schema',
    'name',
    'title',
    'description',
    'inputSchema',
    'annotations',
    'type',
    'properties',
    'required',
    'enum',
    'items',
    'additionalProperties',
];

/**
 * Reorders an object's top-level keys to match FIELD_ORDER, then appends the rest.
 */
function reorderObject<T extends Record<string, unknown>>(obj: T, keyOrder: string[]): T {
    const ordered: Record<string, unknown> = {};
    // 1. Add keys in specified order
    for (const key of keyOrder) {
        if (key in obj) ordered[key] = obj[key as keyof T];
    }
    // 2. Add any remaining keys
    for (const key of Object.keys(obj)) {
        if (!keyOrder.includes(key)) ordered[key] = obj[key as keyof T];
    }
    return ordered as T;
}

/**
* Writes JSON-Schema files for a list of TypeScript types.
* @param program     ts.Program instance
* @param items       List of shapes to generate
* @param outBaseDir  Output directory
*/
function generateSchemas(
    program: TJS.Program,
    items: TypeToSchema[],
    outBaseDir: string,
) {
    // Cleanup the output directory (skip root – root cleaned once externally)
    if (outBaseDir !== OUT_SCHEMAS_BASE && fs.existsSync(outBaseDir)) {
        fs.rmSync(outBaseDir, { recursive: true, force: true });
    }
    fs.mkdirSync(outBaseDir, { recursive: true });

    // Loop through the items and generate the schemas
    for (const { subdir, typeName, fileName, useRef, name, annotations, transform } of items) {
        try {
            // Clone the base settings and override 'ref' per schema
            const schemaSettings = { ...baseSettings, ref: useRef ?? baseSettings.ref };
            let schema = TJS.generateSchema(program, typeName, schemaSettings) as Record<string, unknown>;

            if (!schema) {
                console.error(`⚠️  Failed to generate schema for ${typeName}`);
                failureCount++;
                continue;
            }

            // Override root title if provided
            if (name) {
                (schema as Record<string, any>).name = name;
            }
            // Clean up the schema
            stripTitles(schema, true);
            stripBoilerplate(schema);
            // Add top-level annotations if provided
            if (annotations) {
                (schema as Record<string, any>).annotations = annotations;
            }
            // Apply any custom transform hook
            if (transform) {
                schema = transform(schema);
            }
            // Reorder top-level fields before serialization
            schema = reorderObject(schema, FIELD_ORDER);

            // Write the schema to the output directory, also creating the typeName directory if it doesn't exist
            const outputPath = subdir ? path.join(outBaseDir, subdir, `${fileName}.json`) : path.join(outBaseDir, `${fileName}.json`);
            if (subdir) fs.mkdirSync(path.join(outBaseDir, subdir), { recursive: true });
            fs.writeFileSync(
                outputPath,
                JSON.stringify(schema, null, 2),
            );
            console.info(`  • ${outputPath}`);
        } catch (err) {
            failureCount++;
            console.error(`⚠️  ${subdir ? `${subdir}/` : ""}${fileName} – ${err instanceof Error ? err.message : String(err)}`);
        }
    }
}

/* ─────────────────────── 2. OpenAI-tool Schemas ────────────────────────── */

/**
 * These helper interfaces *wrap* OpenAI's SDK discriminated-union so that
 * `typescript-json-schema` can emit independent schemas per tool type.
 *
 * **NOTE:**  We keep the actual definitions in `shapes.ts` so that both the
 * runtime app **and** this script draw from the same single source of truth.
 *
 * ```ts
 * import type OpenAI from "openai";
 * type OpenAITool = OpenAI.Responses.Tool;
 * export interface CodeInterpreterSpec   extends Extract<OpenAITool,{type:"code_interpreter"}> {}
 * export interface ImageGenerationSpec   extends Extract<OpenAITool,{type:"image_generation"}> {}
 * // … etc …
 * ```
 */
const OPENAI_TOOL_ITEMS: Array<{ variant: string; typeName: string; fileName: string }> = [
    { variant: "code_interpreter", typeName: "CodeInterpreterSpec", fileName: "code_interpreter" },
    { variant: "image_generation", typeName: "ImageGenerationSpec", fileName: "image_generation" },
    { variant: "mcp", typeName: "MCPSpec", fileName: "mcp" },
    { variant: "file_search", typeName: "FileSearchSpec", fileName: "file_search" },
    { variant: "web_search_preview", typeName: "WebSearchPreviewSpec", fileName: "web_search_preview" },
    { variant: "computer_preview", typeName: "ComputerUsePreviewSpec", fileName: "computer_use_preview" },
];


/* ────────────────────────────── Main flow ──────────────────────────────── */

console.info("🔄  Building TypeScript program…");
const program = TJS.getProgramFromFiles(
    [
        IN_TYPES_RESOURCES,
        IN_TYPES_SERVICES,
        IN_TYPES_TOOLS,
    ],
    compilerOptions,
    monorepoRoot,
);

// Clean the root schemas directory exactly once
if (fs.existsSync(OUT_SCHEMAS_BASE)) {
    fs.rmSync(OUT_SCHEMAS_BASE, { recursive: true, force: true });
}
fs.mkdirSync(OUT_SCHEMAS_BASE, { recursive: true });

// 1️⃣  Resource schemas
console.info("\n📦 Generating resource schemas:");
generateSchemas(
    program,
    RESOURCE_SCHEMAS,
    OUT_DEFINE_TOOL_BASE,
);

// 2️⃣  MCP schemas
console.info("\n📦 Generating MCP tool-schemas:");
generateSchemas(
    program,
    MCP_TOOL_SCHEMAS,
    OUT_SCHEMAS_BASE,
);

// 3️⃣  AI service tool schemas
console.info("\n🤖 Generating OpenAI tool-schemas:");
generateSchemas(program, OPENAI_TOOL_ITEMS, OUT_SERVICES_BASE);

/* ────────────────────────────── Epilogue ───────────────────────────────── */

if (failureCount > 0) {
    console.info(`\n❌  Schema generation complete. Failures: ${failureCount}.`);
} else {
    console.info("\n✅  Schema generation complete.");
}