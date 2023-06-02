/**
 * Converts query/mutation data (which is set up to be type-safe and reduce code duplication) into either graphql tags 
 * or PartialGraphQLInfo objects, depending on whether the desired endpoint is a graphql endpoint or a rest endpoint.
 * This is done during build to reduce runtime computation.
 */
import { resolveGQLInfo } from "@local/shared";
import fs from "fs";
import { DocumentNode, FieldNode, FragmentDefinitionNode, GraphQLResolveInfo, OperationDefinitionNode, parse } from "graphql";
import { injectTypenames } from "../../../../server/src/builders/injectTypenames";
import { FormatMap } from "../../../../server/src/models/format";
import { endpoints } from "./endpoints";

// Specify whether to generate graphql-tag strings, PartialGraphQLInfo objects, or both. 
// Tags strings are more flexible, but PartialGraphQLInfo objects are more efficient.
type Target = "graphql" | "rest" | "both";
const target: Target = "both";

console.info("Generating graphql-tag strings for endpoints...");

// Create the output folders if they doesn't exist
const outputFolder = "../shared/src/api/generated";
const fragmentsFolder = `${outputFolder}/fragments`;
const endpointsFolder = `${outputFolder}/endpoints`;
const restFolder = `${outputFolder}/rest`;
const createFolder = (folder: string) => {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`);
        fs.mkdirSync(folder);
    }
};
createFolder(outputFolder);
if (["graphql", "both"].includes(target)) {
    createFolder(fragmentsFolder);
    createFolder(endpointsFolder);
}
if (["rest", "both"].includes(target)) {
    createFolder(restFolder);
}

// Store GraphQL data, which is used to generate both the graphql-tag strings and the PartialGraphQLInfo objects
const allFragments: { [name: string]: string } = {};
const allEndpoints: { [name: string]: string } = {};

// Find GraphQL endpoints
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
        // Create full endpoint string and file name
        const endpointString = `export const ${objectType}${endpointName[0].toUpperCase() + endpointName.slice(1)} = gql\`${tag}\`;\n\n`;
        const outputFile = `${objectType}_${endpointName}`;
        // Store in total endpoint object
        allEndpoints[outputFile] = `${importsString}\n\n${endpointString}`;
    }
}

// Find GraphQL fragments
// Note: These do not use gql tags because they are always used in other gql tags. 
// If we made these gql tags, we'd have to import nested fragments, which might 
// cause duplicate imports.
for (const [name, fragment] of Object.entries(allFragments)) {
    allFragments[name] = `export const ${name} = \`${fragment}\`;\n`;
}

// If target is "graphql" or "both", write the endpoints and fragments to files
if (["graphql", "both"].includes(target)) {
    // Store endpoints
    for (const [name, endpoint] of Object.entries(allEndpoints)) {
        const outputPath = `${outputFolder}/endpoints/${name}.ts`;
        console.log(`generating endpoint ${name}...`);
        // Write to file
        fs.writeFileSync(outputPath, endpoint);
    }
    // Store fragments
    for (const [name, fragment] of Object.entries(allFragments)) {
        const outputPath = `${outputFolder}/fragments/${name}.ts`;
        console.log(`generating fragment ${name}...`);
        // Write to file
        fs.writeFileSync(outputPath, fragment);
    }
    // Create index.ts files
    console.info("Generating index.ts files for endpoints and fragments...");
    if (["graphql", "both"].includes(target)) {
        // Create index.ts for endpoints
        let indexContent = Object.keys(allEndpoints).map(name => `export * from "./${name}";`).join("\n");
        fs.writeFileSync(`${endpointsFolder}/index.ts`, indexContent);
        // Create index.ts for fragments
        indexContent = Object.keys(allFragments).map(name => `export * from "./${name}";`).join("\n");
        fs.writeFileSync(`${fragmentsFolder}/index.ts`, indexContent);
    }
}
// Otherwise, delete the endpoints and fragments folders
else {
    fs.rmdirSync(endpointsFolder, { recursive: true });
    fs.rmdirSync(fragmentsFolder, { recursive: true });
}

// If target is "rest", convert endpoint tags to PartialGraphQLInfo objects, 
// for use in REST endpoints
if (["rest", "both"].includes(target)) {
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
    // Loop through allEndpoints
    for (const [name, endpoint] of Object.entries(allEndpoints)) {
        console.log(`Generating GraphQLResolveInfo for ${name}...`);
        // Extract the gql tag using a regex
        const gqlTagMatch = endpoint.match(/gql`([\s\S]*?)`;/);
        if (!gqlTagMatch) {
            console.error(`No gql tag found in ${name}`);
            continue;
        }
        let gqlTag = gqlTagMatch[1];
        // Replace fragment placeholders with actual fragment definitions
        let fragmentMatch;
        const fragmentPlaceholderRegex = /\$\{(\w+)\}/g;
        while ((fragmentMatch = fragmentPlaceholderRegex.exec(gqlTag)) !== null) {
            const fragmentName = fragmentMatch[1];
            const fragmentContent = allFragments[fragmentName];
            const fragment = (fragmentContent.match(/`([\s\S]*?)`;/) || [])[1];
            if (fragment) {
                gqlTag = gqlTag.replace(fragmentMatch[0], fragment);
            } else {
                console.error(`No fragment found in ${fragmentName}.ts`);
            }
        }
        // Parse the gql tag into a DocumentNode
        const documentNode: DocumentNode = parse(gqlTag);
        // Generate the GraphQLResolveInfo object
        let resolveInfo: any = gqlToGraphQLResolveInfo(documentNode, name.replace(".ts", ""));
        // Attempt to convert to PartialGraphQLResolveInfo, a shorter version of GraphQLResolveInfo
        const __typename = name.split("_")[0][0].toUpperCase() + name.split("_")[0].slice(1);
        if (__typename in FormatMap) {
            resolveInfo = resolveGQLInfo(resolveInfo);
            resolveInfo = injectTypenames(resolveInfo, FormatMap[__typename].gqlRelMap);
        }
        // Stringify and format the resolveInfo object
        const stringified = JSON.stringify(resolveInfo, null, 2);
        // Write the GraphQLResolveInfo object to a new file in the rest directory
        const restFilePath = `${restFolder}/${name}.ts`;
        fs.writeFileSync(restFilePath, `export const ${name.replace(".ts", "")} = ${stringified} as const;\n`);
        restFiles.push(name.replace(".ts", ""));
    }
    // Create index.ts for rest
    console.info("Generating index.ts for rest...");
    const indexContentRest = restFiles.map(file => `export * from "./${file}";`).join("\n");
    fs.writeFileSync(`${restFolder}/index.ts`, indexContentRest);
}
// Otherwise, delete the rest folder
else {
    fs.rmdirSync(restFolder, { recursive: true });
}

// Create index.ts for generated folder
const indexArray = ["graphqlTypes"];
if (["rest", "both"].includes(target)) {
    indexArray.push("rest");
}
if (["graphql", "both"].includes(target)) {
    indexArray.push("endpoints");
    indexArray.push("fragments");
}
const indexContent = indexArray.map(name => `export * from "./${name}";`).join("\n");
fs.writeFileSync(`${outputFolder}/index.ts`, indexContent);

console.info("Finished generating graphql-tag strings for endpoints🚀");
