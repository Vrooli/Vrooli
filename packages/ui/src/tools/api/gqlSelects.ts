/**
 * Converts query/mutation data (which is set up to be type-safe and reduce code duplication) into either graphql tags 
 * or PartialGraphQLInfo objects, depending on whether the desired endpoint is a graphql endpoint or a rest endpoint.
 * This is done during build to reduce runtime computation.
 */
import fs from "fs";
import { DocumentNode, FieldNode, FragmentDefinitionNode, GraphQLResolveInfo, OperationDefinitionNode, parse } from "graphql";
import { dirname } from "path";
import { fileURLToPath } from "url";
import { endpoints } from "./endpoints";

type Target = "graphql" | "rest" | "both";
const target: Target = "both";

console.info("Generating graphql-tag strings for endpoints...");

// Step 1: Create output folders
// Create the output folders if they doesn't exist
const outputFolder = "../shared/src/api/generated";
const fragmentsFolder = `${outputFolder}/fragments`;
const endpointsFolder = `${outputFolder}/endpoints`;
const restFolder = `${outputFolder}/rest`;
for (const folder of [outputFolder, fragmentsFolder, endpointsFolder, restFolder]) {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`);
        fs.mkdirSync(folder);
    }
}

// Step 2: Find data and write endpoints to files
// Initialize fragments list to store all fragment definitions
const allFragments: { [name: string]: string } = {};
const endpointFiles: string[] = [];
// Unlazy each endpoint property and write it to a separate file
for (const objectType of Object.keys(endpoints)) {
    const endpointGroup = await (endpoints as any)[objectType]();
    console.log(`generating endpoints for ${objectType}...`);
    for (const endpointName of Object.keys(endpointGroup)) {
        // Get the endpoint data
        const { fragments, tag } = await endpointGroup[endpointName] as { fragments: [string, string][], tag: string };
        // Add the fragments to the total fragment object
        for (const [name, fragment] of fragments) {
            allFragments[name] = fragment;
        }
        // Calculate imports, startig with the gql import
        let importsString = "import gql from \"graphql-tag\";";
        // Add import for each fragment
        for (const [fragmentName] of fragments) {
            importsString += `\nimport { ${fragmentName} } from "../fragments/${fragmentName}";`;
        }
        // Write imports and endpoint to file
        const endpointString = `export const ${objectType}${endpointName[0].toUpperCase() + endpointName.slice(1)} = gql\`${tag}\`;\n\n`;
        const outputPath = `${outputFolder}/endpoints/${objectType}_${endpointName}.ts`;
        fs.writeFileSync(outputPath, `${importsString}\n\n${endpointString}`);
        endpointFiles.push(`${objectType}_${endpointName}`);
    }
}

// Step 3: Write fragments to files
// Note: These do not use gql tags because they are always used in other gql tags. 
// If we made these gql tags, we'd have to import nested fragments, which might 
// cause duplicate imports.
for (const [name, fragment] of Object.entries(allFragments)) {
    const outputPath = `${outputFolder}/fragments/${name}.ts`;
    console.log(`generating fragment ${name}...`);
    // Write to file
    fs.writeFileSync(outputPath, `export const ${name} = \`${fragment}\`;\n`);
}

// Step 4: If target is "rest", convert endpoint tags to GraphQLResolveInfo objects, 
// for use in REST endpoints
if (target === "rest" || target === "both") {
    // Initialize restFiles list to store all rest files names
    const restFiles: string[] = [];
    // Define function to convert gql tags to GraphQLResolveInfo objects
    const gqlToGraphQLResolveInfo = (query: DocumentNode, path: string): GraphQLResolveInfo => {
        const operation: OperationDefinitionNode | undefined = query.definitions
            .find(({ kind }) => kind === "OperationDefinition") as OperationDefinitionNode;

        const fragmentDefinitions = query.definitions
            .filter(({ kind }) => kind === "FragmentDefinition") as FragmentDefinitionNode[];

        const fragments: { [key: string]: FragmentDefinitionNode } = fragmentDefinitions
            .reduce((result, current: FragmentDefinitionNode) => ({
                ...result,
                [current.name.value]: current,
            }), {});

        const fieldNodes: FieldNode[] = operation?.selectionSet.selections as FieldNode[];

        const resolveInfo: GraphQLResolveInfo = {
            cacheControl: {
                cacheHint: {},
                // eslint-disable-next-line @typescript-eslint/no-empty-function
                setCacheHint: () => { },
                cacheHintFromType: () => ({}),
            },
            fieldName: fieldNodes[0]?.name.value || "",
            fieldNodes,
            returnType: null,
            parentType: null,
            schema: null,
            fragments,
            rootValue: {},
            operation,
            variableValues: {},
            path: {
                prev: undefined,
                key: path,
            },
        } as any;
        return resolveInfo;
    };
    // Read through the files in the endpoints directory
    const files = fs.readdirSync(endpointsFolder);
    for (const file of files) {
        // Skip index.ts
        if (file === "index.ts") {
            continue;
        }
        console.log(`Generating GraphQLResolveInfo for ${file}...`);

        // Read the file as a string
        const fileContent = fs.readFileSync(`${endpointsFolder}/${file}`, "utf8");

        // Extract the gql tag using a regex
        const gqlTagMatch = fileContent.match(/gql`([\s\S]*?)`;/);
        if (!gqlTagMatch) {
            console.error(`No gql tag found in ${file}`);
            continue;
        }
        let gqlTag = gqlTagMatch[1];

        // Replace fragment placeholders with actual fragment definitions
        let fragmentMatch;
        const fragmentPlaceholderRegex = /\$\{(\w+)\}/g;
        while ((fragmentMatch = fragmentPlaceholderRegex.exec(gqlTag)) !== null) {
            const fragmentName = fragmentMatch[1];
            const fragmentFileContent = fs.readFileSync(`${fragmentsFolder}/${fragmentName}.ts`, "utf8");
            const fragment = (fragmentFileContent.match(/`([\s\S]*?)`;/) || [])[1];
            if (fragment) {
                gqlTag = gqlTag.replace(fragmentMatch[0], fragment);
            } else {
                console.error(`No fragment found in ${fragmentName}.ts`);
            }
        }

        // Parse the gql tag into a DocumentNode
        const documentNode: DocumentNode = parse(gqlTag);

        // Generate the GraphQLResolveInfo object
        const resolveInfo: GraphQLResolveInfo = gqlToGraphQLResolveInfo(documentNode, file.replace(".ts", ""));

        // Stringify and format the resolveInfo object
        const resolveInfoStr = JSON.stringify(resolveInfo, null, 2);

        // Write the GraphQLResolveInfo object to a new file in the rest directory
        const restFilePath = `${restFolder}/${file}`;
        fs.writeFileSync(restFilePath, `export const ${file.replace(".ts", "")} = ${resolveInfoStr};\n`);

        restFiles.push(file.replace(".ts", ""));
    }
    // Create index.ts for rest
    console.info("Generating index.ts for rest...");
    const indexContentRest = restFiles.map(file => `export * from "./${file}";`).join("\n");
    fs.writeFileSync(`${restFolder}/index.ts`, indexContentRest);
    // If not targeting graphql, delete the endpoints and fragments folders
    if (target === "rest") {
        fs.rmdirSync(endpointsFolder, { recursive: true });
        fs.rmdirSync(fragmentsFolder, { recursive: true });
    }
}

// Step 5: Write index.ts files
console.info("Generating index.ts files...");
// Create index.ts for endpoints
let indexContent = endpointFiles.map(file => `export * from "./${file}";`).join("\n");
fs.writeFileSync(`${endpointsFolder}/index.ts`, indexContent);

// Create index.ts for fragments
indexContent = Object.keys(allFragments).map(name => `export * from "./${name}";`).join("\n");
fs.writeFileSync(`${fragmentsFolder}/index.ts`, indexContent);

// Create index.ts for generated folder
const indexArray = ["graphqlTypes"];
if (target === "rest" || target === "both") {
    indexArray.push("rest");
}
if (target === "graphql" || target === "both") {
    indexArray.push("endpoints");
    indexArray.push("fragments");
}
indexContent = indexArray.map(name => `export * from "./${name}";`).join("\n");
fs.writeFileSync(`${outputFolder}/index.ts`, indexContent);

console.info("Finished generating graphql-tag strings for endpoints🚀");
