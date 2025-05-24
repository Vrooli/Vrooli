import * as fs from "fs";
import * as path from "path";
import ts from "typescript"; // Import the 'ts' namespace for enums
import * as TJS from "typescript-json-schema";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// --- Helper functions ---
function stripTitles(node: any, isRoot = false): void {
    if (!node || typeof node !== "object") return;

    // Delete the title unless we're at the root
    if (!isRoot && "title" in node) delete node.title;

    // Recursively walk through child containers
    if (Array.isArray(node)) {
        node.forEach((item) => stripTitles(item, false));
    } else {
        Object.values(node).forEach((val) => stripTitles(val, false));
    }
}

function stripBoilerplate(node: unknown): void {
    if (Array.isArray(node)) {
        node.forEach(stripBoilerplate);
        return;
    }
    if (node && typeof node === "object") {
        const o = node as Record<string, unknown>;

        // ── 1.  Kill empty defaultProperties arrays ──────────────────────
        if (
            "defaultProperties" in o &&
            Array.isArray(o.defaultProperties) &&
            o.defaultProperties.length === 0
        ) {
            delete o.defaultProperties;
        }

        // ── 2.  Kill explicit `additionalProperties: false` ──────────────
        if (o.additionalProperties === false) {
            delete o.additionalProperties;
        }

        // Recurse into children
        Object.values(o).forEach(stripBoilerplate);
    }
}
// --- End Helper functions ---

// More robust path finding for project roots
const packageServerRoot = path.resolve(__dirname, "../../..");
const monorepoRoot = path.resolve(packageServerRoot, "..");
const shapesFile = path.resolve(packageServerRoot, "src/__tools/mcp/shapes.ts");
const outputDirBase = path.resolve(packageServerRoot, "src/services/mcp/schemas");

// --- Configuration ---
const settings: TJS.PartialArgs = {
    required: true,
    titles: true,
    defaultProps: true,
    validationKeywords: ["format", "pattern", "minLength", "maxLength", "minimum", "maximum", "enum", "default", "items", "properties"],
    ref: false, // Adding refs makes the schemas slightly more complex, which isn't great for AI agents using them
    noExtraProps: true,
    ignoreErrors: true,
};

// Define a NEW, comprehensive set of CompilerOptions for TJS, relative to monorepoRoot
const compilerOptionsForTJS: TJS.CompilerOptions = {
    target: ts.ScriptTarget.ESNext,
    module: ts.ModuleKind.ESNext,
    moduleResolution: ts.ModuleResolutionKind.NodeNext,
    noEmit: true,
    noEmitOnError: true,
    esModuleInterop: true,
    resolveJsonModule: true,
    allowSyntheticDefaultImports: true,
    strictNullChecks: true,
    skipLibCheck: true, // IMPORTANT: To avoid d.ts conflicts
    allowJs: true,
    jsx: ts.JsxEmit.ReactJSX,
    downlevelIteration: true,

    baseUrl: ".", // Relative to monorepoRoot (which will be TJS.getProgramFromFiles basePath)
    rootDir: ".", // All source files are under monorepoRoot
    paths: {
        "@local/shared": ["packages/shared/src"],      // Path from monorepoRoot
        "@local/shared/*": ["packages/shared/src/*"],   // Path from monorepoRoot
    },

    lib: ["esnext", "dom", "dom.iterable"], // Ensure esnext for modern features

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

// Define which types to generate schemas for.
// This structure helps map variant+op to a specific TypeScript interface name.
const schemasToGenerate: Array<{ variant: string; op: "add" | "update" | "find"; typeName: string; outputName?: string; useRef?: boolean }> = [
    // RoutineMultiStep
    { variant: "RoutineMultiStep", op: "add", typeName: "RoutineMultiStepAddAttributes", useRef: true },
    { variant: "RoutineMultiStep", op: "update", typeName: "RoutineMultiStepUpdateAttributes", useRef: true },
    { variant: "RoutineMultiStep", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineInternalAction
    { variant: "RoutineInternalAction", op: "add", typeName: "RoutineInternalActionAddAttributes", useRef: true },
    { variant: "RoutineInternalAction", op: "update", typeName: "RoutineInternalActionUpdateAttributes", useRef: true },
    { variant: "RoutineInternalAction", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineApi
    { variant: "RoutineApi", op: "add", typeName: "RoutineApiAddAttributes", useRef: true },
    { variant: "RoutineApi", op: "update", typeName: "RoutineApiUpdateAttributes", useRef: true },
    { variant: "RoutineApi", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineCode
    { variant: "RoutineCode", op: "add", typeName: "RoutineCodeAddAttributes", useRef: true },
    { variant: "RoutineCode", op: "update", typeName: "RoutineCodeUpdateAttributes", useRef: true },
    { variant: "RoutineCode", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineData
    { variant: "RoutineData", op: "add", typeName: "RoutineDataAddAttributes", useRef: true },
    { variant: "RoutineData", op: "update", typeName: "RoutineDataUpdateAttributes", useRef: true },
    { variant: "RoutineData", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineGenerate
    { variant: "RoutineGenerate", op: "add", typeName: "RoutineGenerateAddAttributes", useRef: true },
    { variant: "RoutineGenerate", op: "update", typeName: "RoutineGenerateUpdateAttributes", useRef: true },
    { variant: "RoutineGenerate", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineInformational
    { variant: "RoutineInformational", op: "add", typeName: "RoutineInformationalAddAttributes", useRef: true },
    { variant: "RoutineInformational", op: "update", typeName: "RoutineInformationalUpdateAttributes", useRef: true },
    { variant: "RoutineInformational", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineSmartContract
    { variant: "RoutineSmartContract", op: "add", typeName: "RoutineSmartContractAddAttributes", useRef: true },
    { variant: "RoutineSmartContract", op: "update", typeName: "RoutineSmartContractUpdateAttributes", useRef: true },
    { variant: "RoutineSmartContract", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // RoutineWeb
    { variant: "RoutineWeb", op: "add", typeName: "RoutineWebAddAttributes", useRef: true },
    { variant: "RoutineWeb", op: "update", typeName: "RoutineWebUpdateAttributes", useRef: true },
    { variant: "RoutineWeb", op: "find", typeName: "ResourceFindFilters", outputName: "filters", useRef: true },
    // Bot
    { variant: "Bot", op: "add", typeName: "BotAddAttributes" },
    { variant: "Bot", op: "update", typeName: "BotUpdateAttributes" },
    { variant: "Bot", op: "find", typeName: "BotFindFilters", outputName: "filters" },
    // Team
    { variant: "Team", op: "add", typeName: "TeamAddAttributes" },
    { variant: "Team", op: "update", typeName: "TeamUpdateAttributes" },
    { variant: "Team", op: "find", typeName: "TeamFindFilters", outputName: "filters" },
    // StandardPrompt
    { variant: "StandardPrompt", op: "add", typeName: "StandardPromptAddAttributes", useRef: true },
    { variant: "StandardPrompt", op: "update", typeName: "StandardPromptUpdateAttributes", useRef: true },
    { variant: "StandardPrompt", op: "find", typeName: "StandardPromptFindFilters", outputName: "filters", useRef: true },
    // StandardDataStructure
    { variant: "StandardDataStructure", op: "add", typeName: "StandardDataStructureAddAttributes", useRef: true },
    { variant: "StandardDataStructure", op: "update", typeName: "StandardDataStructureUpdateAttributes", useRef: true },
    { variant: "StandardDataStructure", op: "find", typeName: "StandardDataStructureFindFilters", outputName: "filters", useRef: true },
    // Project
    { variant: "Project", op: "add", typeName: "ProjectAddAttributes" },
    { variant: "Project", op: "update", typeName: "ProjectUpdateAttributes" },
    { variant: "Project", op: "find", typeName: "ProjectFindFilters", outputName: "filters" },
    // Note
    { variant: "Note", op: "add", typeName: "NoteAddAttributes" },
    { variant: "Note", op: "update", typeName: "NoteUpdateAttributes" },
    { variant: "Note", op: "find", typeName: "NoteFindFilters", outputName: "filters" },
    // ExternalData
    { variant: "ExternalData", op: "add", typeName: "ExternalDataAddAttributes" },
    { variant: "ExternalData", op: "update", typeName: "ExternalDataUpdateAttributes" },
    { variant: "ExternalData", op: "find", typeName: "ExternalDataFindFilters", outputName: "filters" },
];

console.info(`Generating ${schemasToGenerate.length} MCP schemas...`);

// Cleanup the output directory
if (fs.existsSync(outputDirBase)) {
    fs.rmSync(outputDirBase, { recursive: true, force: true }); // Deletes the directory and its contents
}
fs.mkdirSync(outputDirBase, { recursive: true });

// --- Generation Logic (starts after cleanup) ---

const program = TJS.getProgramFromFiles([shapesFile], compilerOptionsForTJS, monorepoRoot);

// Ensure the base output directory exists (though cleanup should have recreated it)
if (!fs.existsSync(outputDirBase)) {
    fs.mkdirSync(outputDirBase, { recursive: true });
}

for (const { variant, op, typeName, outputName, useRef } of schemasToGenerate) {
    try {
        // Clone the base settings and override 'ref' per schema
        const schemaSettings = { ...settings, ref: useRef ?? settings.ref };
        const schema = TJS.generateSchema(program, typeName, schemaSettings);
        if (!schema) {
            console.error(`\tERROR: Could not generate schema for ${typeName} (schema is null/undefined).`);
            continue;
        }

        stripTitles(schema, true);
        stripBoilerplate(schema);

        const variantDir = path.join(outputDirBase, variant);
        if (!fs.existsSync(variantDir)) {
            fs.mkdirSync(variantDir, { recursive: true });
        }

        // For 'add'/'update', the schema describes 'attributes'. For 'find', it describes 'filters'.
        // The 'delete' op typically only needs an 'id', already in the base schema, so usually no specific schema needed.
        const schemaFileName = `${op}_${outputName || "attributes"}.json`;
        const outputPath = path.join(variantDir, schemaFileName);

        fs.writeFileSync(outputPath, JSON.stringify(schema, null, 2));

    } catch (e: any) {
        console.error(`\tERROR generating schema for ${typeName}: ${e.message || e}`);
    }
}

console.info("✅ Schema generation complete.");
