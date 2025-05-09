import * as fs from 'fs';
import * as path from 'path';
import ts from 'typescript'; // Import the 'ts' namespace for enums
import * as TJS from 'typescript-json-schema';
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// More robust path finding for project roots
const packageServerRoot = path.resolve(__dirname, '../../..');
const monorepoRoot = path.resolve(packageServerRoot, '..');
const shapesFile = path.resolve(packageServerRoot, 'src/__tools/mcp/shapes.ts');
const outputDirBase = path.resolve(packageServerRoot, 'src/services/mcp/schemas');

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
        "./packages/shared/src/types"            // Custom types in shared package
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
const schemasToGenerate: Array<{ variant: string; op: 'add' | 'update' | 'find'; typeName: string; outputName?: string }> = [
    // RoutineMultiStep
    { variant: 'RoutineMultiStep', op: 'add', typeName: 'RoutineMultiStepAddAttributes' },
    { variant: 'RoutineMultiStep', op: 'update', typeName: 'RoutineMultiStepUpdateAttributes' },
    { variant: 'RoutineMultiStep', op: 'find', typeName: 'RoutineMultiStepFindFilters', outputName: 'filters' },
    // RoutineInternalAction
    { variant: 'RoutineInternalAction', op: 'add', typeName: 'RoutineInternalActionAddAttributes' },
    { variant: 'RoutineInternalAction', op: 'update', typeName: 'RoutineInternalActionUpdateAttributes' },
    { variant: 'RoutineInternalAction', op: 'find', typeName: 'RoutineInternalActionFindFilters', outputName: 'filters' },
    // RoutineApi
    { variant: 'RoutineApi', op: 'add', typeName: 'RoutineApiAddAttributes' },
    { variant: 'RoutineApi', op: 'update', typeName: 'RoutineApiUpdateAttributes' },
    { variant: 'RoutineApi', op: 'find', typeName: 'RoutineApiFindFilters', outputName: 'filters' },
    // RoutineCode
    { variant: 'RoutineCode', op: 'add', typeName: 'RoutineCodeAddAttributes' },
    { variant: 'RoutineCode', op: 'update', typeName: 'RoutineCodeUpdateAttributes' },
    { variant: 'RoutineCode', op: 'find', typeName: 'RoutineCodeFindFilters', outputName: 'filters' },
    // RoutineData
    { variant: 'RoutineData', op: 'add', typeName: 'RoutineDataAddAttributes' },
    { variant: 'RoutineData', op: 'update', typeName: 'RoutineDataUpdateAttributes' },
    { variant: 'RoutineData', op: 'find', typeName: 'RoutineDataFindFilters', outputName: 'filters' },
    // RoutineGenerate
    { variant: 'RoutineGenerate', op: 'add', typeName: 'RoutineGenerateAddAttributes' },
    { variant: 'RoutineGenerate', op: 'update', typeName: 'RoutineGenerateUpdateAttributes' },
    { variant: 'RoutineGenerate', op: 'find', typeName: 'RoutineGenerateFindFilters', outputName: 'filters' },
    // RoutineInformational
    { variant: 'RoutineInformational', op: 'add', typeName: 'RoutineInformationalAddAttributes' },
    { variant: 'RoutineInformational', op: 'update', typeName: 'RoutineInformationalUpdateAttributes' },
    { variant: 'RoutineInformational', op: 'find', typeName: 'RoutineInformationalFindFilters', outputName: 'filters' },
    // RoutineSmartContract
    { variant: 'RoutineSmartContract', op: 'add', typeName: 'RoutineSmartContractAddAttributes' },
    { variant: 'RoutineSmartContract', op: 'update', typeName: 'RoutineSmartContractUpdateAttributes' },
    { variant: 'RoutineSmartContract', op: 'find', typeName: 'RoutineSmartContractFindFilters', outputName: 'filters' },
    // RoutineWeb
    { variant: 'RoutineWeb', op: 'add', typeName: 'RoutineWebAddAttributes' },
    { variant: 'RoutineWeb', op: 'update', typeName: 'RoutineWebUpdateAttributes' },
    { variant: 'RoutineWeb', op: 'find', typeName: 'RoutineWebFindFilters', outputName: 'filters' },
    // Bot
    { variant: 'Bot', op: 'add', typeName: 'BotAddAttributes' },
    { variant: 'Bot', op: 'update', typeName: 'BotUpdateAttributes' },
    { variant: 'Bot', op: 'find', typeName: 'BotFindFilters', outputName: 'filters' },
    // Team
    { variant: 'Team', op: 'add', typeName: 'TeamAddAttributes' },
    { variant: 'Team', op: 'update', typeName: 'TeamUpdateAttributes' },
    { variant: 'Team', op: 'find', typeName: 'TeamFindFilters', outputName: 'filters' },
    // StandardPrompt
    { variant: 'StandardPrompt', op: 'add', typeName: 'StandardPromptAddAttributes' },
    { variant: 'StandardPrompt', op: 'update', typeName: 'StandardPromptUpdateAttributes' },
    { variant: 'StandardPrompt', op: 'find', typeName: 'StandardPromptFindFilters', outputName: 'filters' },
    // StandardDataStructure
    { variant: 'StandardDataStructure', op: 'add', typeName: 'StandardDataStructureAddAttributes' },
    { variant: 'StandardDataStructure', op: 'update', typeName: 'StandardDataStructureUpdateAttributes' },
    { variant: 'StandardDataStructure', op: 'find', typeName: 'StandardDataStructureFindFilters', outputName: 'filters' },
    // Project
    { variant: 'Project', op: 'add', typeName: 'ProjectAddAttributes' },
    { variant: 'Project', op: 'update', typeName: 'ProjectUpdateAttributes' },
    { variant: 'Project', op: 'find', typeName: 'ProjectFindFilters', outputName: 'filters' },
    // Note
    { variant: 'Note', op: 'add', typeName: 'NoteAddAttributes' },
    { variant: 'Note', op: 'update', typeName: 'NoteUpdateAttributes' },
    { variant: 'Note', op: 'find', typeName: 'NoteFindFilters', outputName: 'filters' },
    // ExternalData
    { variant: 'ExternalData', op: 'add', typeName: 'ExternalDataAddAttributes' },
    { variant: 'ExternalData', op: 'update', typeName: 'ExternalDataUpdateAttributes' },
    { variant: 'ExternalData', op: 'find', typeName: 'ExternalDataFindFilters', outputName: 'filters' },
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

for (const { variant, op, typeName, outputName } of schemasToGenerate) {
    try {
        const schema = TJS.generateSchema(program, typeName, settings);
        if (!schema) {
            console.error(`\tERROR: Could not generate schema for ${typeName} (schema is null/undefined).`);
            continue;
        }

        const variantDir = path.join(outputDirBase, variant);
        if (!fs.existsSync(variantDir)) {
            fs.mkdirSync(variantDir, { recursive: true });
        }

        // For 'add'/'update', the schema describes 'attributes'. For 'find', it describes 'filters'.
        // The 'delete' op typically only needs an 'id', already in the base schema, so usually no specific schema needed.
        const schemaFileName = `${op}_${outputName || 'attributes'}.json`;
        const outputPath = path.join(variantDir, schemaFileName);

        fs.writeFileSync(outputPath, JSON.stringify(schema, null, 2));

    } catch (e: any) {
        console.error(`\tERROR generating schema for ${typeName}: ${e.message || e}`);
    }
}

console.info("✅ Schema generation complete.");
