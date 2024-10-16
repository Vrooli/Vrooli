/**
 * Converts query/mutation data (which is set up to be type-safe and reduce code duplication) into either graphql tags 
 * or PartialGraphQLInfo objects, depending on whether the desired endpoint is a graphql endpoint or a rest endpoint.
 * This is done during build to reduce runtime computation.
 */
import { resolveGQLInfo, uppercaseFirstLetter } from "@local/shared";
import fs from "fs";
import { DocumentNode, FieldNode, FragmentDefinitionNode, GraphQLResolveInfo, OperationDefinitionNode, parse } from "graphql";
import { injectTypenames } from "../../../../server/src/builders/injectTypenames";
import { FormatMap } from "../../../../server/src/models/formats";
import { endpoints } from "./endpoints";

// Specify whether to generate graphql-tag strings, PartialGraphQLInfo objects, or both. 
// Tags strings are more flexible, but PartialGraphQLInfo objects are more efficient.
type Target = "graphql" | "rest" | "both";
const target: Target = "rest";
// If true, will generate endpoint/method pairs for each endpoint, based on what's defined in `packages/server/src/endpoints/rest`. 
// Only used if `target` is "rest" or "both".
const createEndpointMethodPairs = true;

function createFolder(folder: string) {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`);
        fs.mkdirSync(folder);
    }
}
function deleteFolder(folder: string) {
    if (fs.existsSync(folder)) {
        console.info(`Deleting folder: ${folder}`);
        fs.rmdirSync(folder, { recursive: true });
    }
}
function deleteFile(file: string) {
    if (fs.existsSync(file)) {
        console.info(`Deleting file: ${file}`);
        fs.unlinkSync(file);
    }
}

// Converts gql tags to GraphQLResolveInfo objects
function gqlToGraphQLResolveInfo(query: DocumentNode, path: string): GraphQLResolveInfo {
    const operation: OperationDefinitionNode | undefined = query.definitions
        .find(({ kind }) => kind === "OperationDefinition") as OperationDefinitionNode;
    const fragmentDefinitions = query.definitions
        .filter(({ kind }) => kind === "FragmentDefinition") as FragmentDefinitionNode[];
    const fragments: { [key: string]: FragmentDefinitionNode } = fragmentDefinitions
        .reduce((result, current: FragmentDefinitionNode) => ({
            ...result,
            [current.name.value]: current,
        }), {});

    const operationFieldNodes: FieldNode[] = operation?.selectionSet.selections as FieldNode[];
    // Add fields from fragments to the operation's fieldNodes
    const fragmentFieldNodes: FieldNode[] = fragmentDefinitions.flatMap(fragment => fragment.selectionSet.selections) as FieldNode[];
    const fieldNodes = [...operationFieldNodes, ...fragmentFieldNodes];
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
}

// Converts endpoints to proper names
function endpointToCamelCase(endpoint: string): string {
    // Split string by slash and filter out empty strings and strings containing ':'
    const parts = endpoint.split("/").filter(p => p && !p.includes(":"));

    // If there's only one part, return it
    if (parts.length === 1) {
        return parts[0] as string;
    }

    // Map through the array to construct the camelCase string
    const camelCaseString = parts.map((part, index) => {
        // If it's the first part, return it with the first character lowercase
        if (index === 0) {
            return part.charAt(0).toLowerCase() + part.slice(1);
        }
        // If it's not the first part, return it with the first character capitalized
        return part.charAt(0).toUpperCase() + part.slice(1);
    }).join("");

    return camelCaseString;
}

async function main() {
    // Create the output folders if they doesn't exist
    const sharedOutputFolder = "../shared/src/api/generated";
    const fragmentsFolder = `${sharedOutputFolder}/fragments`;
    const endpointsFolder = `${sharedOutputFolder}/endpoints`;
    const restFolder = "../server/src/endpoints/generated";
    createFolder(sharedOutputFolder);
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
    console.info("Generating graphql-tag strings for endpoints...");
    for (const objectType of Object.keys(endpoints)) {
        const endpointGroup = await (endpoints as any)[objectType]();
        console.info(`generating endpoints for ${objectType}...`);
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
            const endpointString = `export const ${objectType}${uppercaseFirstLetter(endpointName)} = gql\`${tag}\`;\n\n`;
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
            const outputPath = `${sharedOutputFolder}/endpoints/${name}.ts`;
            console.info(`generating endpoint ${name}...`);
            // Write to file
            fs.writeFileSync(outputPath, endpoint);
        }
        // Store fragments
        for (const [name, fragment] of Object.entries(allFragments)) {
            const outputPath = `${sharedOutputFolder}/fragments/${name}.ts`;
            console.info(`generating fragment ${name}...`);
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
        deleteFolder(endpointsFolder);
        deleteFolder(fragmentsFolder);
    }

    // If target is "rest", convert endpoint tags to PartialGraphQLInfo objects, 
    // for use in REST endpoints. 
    // Also generate endpoint/method pairs for each endpoint, if `createEndpointMethodPairs` is true.
    if (["rest", "both"].includes(target)) {
        // Initialize restFiles list to store all rest files names
        const restFiles: string[] = [];
        // Loop through allEndpoints
        for (const [name, endpoint] of Object.entries(allEndpoints)) {
            // Extract the gql tag using a regex
            const gqlTagMatch = endpoint.match(/gql`([\s\S]*?)`;/);
            if (!gqlTagMatch || gqlTagMatch.length < 2) {
                console.error(`No gql tag found in ${name}`);
                continue;
            }
            let gqlTag = gqlTagMatch[1] as string;
            // Replace fragment placeholders with actual fragment definitions
            let fragmentMatch;
            const fragmentPlaceholderRegex = /\$\{(\w+)\}/g;
            while ((fragmentMatch = fragmentPlaceholderRegex.exec(gqlTag)) !== null) {
                const fragmentName = fragmentMatch[1];
                const fragmentContent = allFragments[fragmentName];
                const fragment = (fragmentContent?.match(/`([\s\S]*?)`;/) || [])[1];
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
            const __typename = uppercaseFirstLetter(name.split("_")?.[0] ?? "");
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

        // If createEndpointMethodPairs is true, generate endpoint/method pairs for each endpoint
        if (createEndpointMethodPairs) {
            console.info("Generating endpoint/method pairs...");
            const pairsSrcFolder = "../server/src/endpoints/rest";
            const pairFilePath = `${sharedOutputFolder}/pairs.ts`;

            // Find the name of every file in the pairs folder
            const excludedFiles = ["base", "index", "payment", "root", "types"];
            const restFiles = fs.readdirSync(pairsSrcFolder).map(file => file.replace(".ts", "")).filter(file => !excludedFiles.includes(file));

            // Create backup of pairs file
            if (fs.existsSync(pairFilePath)) {
                fs.copyFileSync(pairFilePath, `${pairFilePath}.bak`);
            }
            // Clear the pairs file
            fs.writeFileSync(pairFilePath, "");

            for (const restFile of restFiles) {
                try {
                    // Import rest file
                    await import(`../../../${pairsSrcFolder}/${restFile}.ts`).then(m => {
                        // Find router
                        const router = m[`${restFile.charAt(0).toUpperCase() + restFile.slice(1)}Rest`];
                        if (!router) {
                            console.error(`No endpoint router found in ${restFile}.ts. If this is not an error, consider adding to to the excludedFiles array.`);
                            return;
                        }
                        // Parse router middleware
                        router.stack.forEach(middleware => {
                            if (middleware.route) {
                                const path = middleware.route.path;
                                const methods = Object.keys(middleware.route.methods);
                                const endpointMethodPairs: {
                                    name: string;
                                    endpoint: string;
                                    method: string;
                                    /** Used as OpenAPI schema operation tag */
                                    tag: string;
                                }[] = [];
                                methods.forEach(method => {
                                    endpointMethodPairs.push({
                                        name: endpointToCamelCase(`${method.toLowerCase()}/${path}`),
                                        endpoint: path,
                                        method: method.toUpperCase(),
                                        tag: restFile,
                                    });
                                });
                                // Append endpoint/method pairs to pairs file
                                const fileContent = endpointMethodPairs.map(pair => `export const ${"endpoint" + pair.name.charAt(0).toUpperCase() + pair.name.slice(1)} = {
    endpoint: "${pair.endpoint}",
    method: "${pair.method}",
    tag: "${pair.tag}",
} as const;\n\n`).join("");
                                fs.appendFileSync(pairFilePath, fileContent);
                            }
                        });
                    });
                } catch (error) {
                    console.error(`Error processing REST file ${restFile}.ts: ${error}`);
                    // Restore pairs file from backup
                    if (fs.existsSync(`${pairFilePath}.bak`)) {
                        fs.copyFileSync(`${pairFilePath}.bak`, pairFilePath);
                    }
                } finally {
                    // Delete pairs file backup
                    deleteFile(`${pairFilePath}.bak`);
                }
            }
        }
    }
    // Otherwise, delete the rest folder and pairs file
    else {
        deleteFolder(restFolder);
        deleteFile(`${sharedOutputFolder}/pairs.ts`);
    }

    // Create index.ts for generated folder
    const indexArray = ["graphqlTypes"];
    if (["rest", "both"].includes(target)) {
        // indexArray.push("rest"); // Stored in the server instead, so not referenced in shared index.ts
        if (createEndpointMethodPairs) {
            indexArray.push("pairs");
        }
    }
    if (["graphql", "both"].includes(target)) {
        indexArray.push("endpoints");
        indexArray.push("fragments");
    }
    const indexContent = indexArray.map(name => `export * from "./${name}";`).join("\n");
    fs.writeFileSync(`${sharedOutputFolder}/index.ts`, indexContent);

    console.info("Finished generating graphql-tag strings for endpoints🚀");
}

main().catch(console.error).finally(() => process.exit());
