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

// --- REMOVED loading and parsing of shared tsconfig --- 
// We will define a single comprehensive CompilerOptions below.

// Helper function to map string values to TS enums (RETAINED, might be useful if we re-introduce tsconfig loading selectively)
function mapStringToTsEnum(value: string | undefined, enumObject: any, defaultValue: any): any {
    if (value && enumObject[value.toUpperCase() as any]) {
        return enumObject[value.toUpperCase() as any];
    }
    if (value && enumObject[value as any]) {
        return enumObject[value as any];
    }
    if (value && value.toLowerCase() === "react-jsx" && enumObject["ReactJSX"]) {
        return enumObject["ReactJSX"];
    }
    return defaultValue;
}

// --- Configuration ---
const settings: TJS.PartialArgs = {
    required: true,
    titles: true,
    defaultProps: true,
    validationKeywords: ["format", "pattern", "minLength", "maxLength", "minimum", "maximum", "enum", "default", "items", "properties"],
    ref: true,
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
    // Team
    { variant: 'Team', op: 'add', typeName: 'McpTeamAddAttributes' },
    { variant: 'Team', op: 'update', typeName: 'McpTeamUpdateAttributes' },
    { variant: 'Team', op: 'find', typeName: 'McpTeamFindFilters', outputName: 'filters' }, // For find, schema is for 'filters'
    // Note
    { variant: 'Note', op: 'add', typeName: 'McpNoteAddAttributes' },
    // ... Add all your variant + op combinations and their corresponding TypeScript interface names
];

// --- Generation Logic ---
console.log(`Generating MCP JSON Schemas from: ${shapesFile}`);
console.log(`Outputting to: ${outputDirBase}`);
console.log(`Using basePath for TS program: ${monorepoRoot}`);
console.log('Using the following explicit CompilerOptions for TJS:', JSON.stringify(compilerOptionsForTJS, (key, value) =>
    typeof value === 'number' && (key === 'target' || key === 'module' || key === 'moduleResolution' || key === 'jsx') ? ts[value] : value, 2));

const program = TJS.getProgramFromFiles([shapesFile], compilerOptionsForTJS, monorepoRoot);

if (!fs.existsSync(outputDirBase)) {
    fs.mkdirSync(outputDirBase, { recursive: true });
}

for (const { variant, op, typeName, outputName } of schemasToGenerate) {
    console.log(`  Generating for: Variant=${variant}, Op=${op}, Type=${typeName}`);
    try {
        const schema = TJS.generateSchema(program, typeName, settings);
        if (!schema) {
            console.error(`    ERROR: Could not generate schema for ${typeName} (schema is null/undefined).`);
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
        console.log(`    SUCCESS: Schema saved to ${outputPath}`);

    } catch (e: any) {
        console.error(`    ERROR generating schema for ${typeName}: ${e.message || e}`);
        // For more detailed error, especially compilation issues:
        // if (e.diagnostics) { // typescript-json-schema might not expose diagnostics directly in catch
        //     e.diagnostics.forEach((diag: any) => console.error(diag.messageText));
        // }
    }
}

console.log("Schema generation complete.");
