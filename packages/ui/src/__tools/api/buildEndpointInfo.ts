/**
 * Converts query/mutation data (which is set up to be type-safe and reduce code duplication) into either graphql tags 
 * or PartialGraphQLInfo objects, depending on whether the desired endpoint is a graphql endpoint or a rest endpoint.
 * This is done during build to reduce runtime computation.
 */
import { injectTypenames } from "@local/server/src/builders/injectTypenames";
import { FormatMap } from "@local/server/src/models/formats";
import { uppercaseFirstLetter } from "@local/shared";
import fs from "fs";
import { DocumentNode, parse } from "graphql";
import { endpoints } from "./endpoints";

// TODO 1 for morning: This script should now only generate information for the UI to call the REST endpoints. Generating prisma select objects should be handled by the server, using a simplified version of the GqlPartial objects that doesn't use __define or __union or anything like that (can still use nav, common, etc.). And shared package script should still generate OpenAPI schema, but in a better way probably. When done, we should make sure that any old references to GraphQL in the 3 API_GENERATE scripts (see build.sh) are removed, and any code with "loc" (indicating old generated graphql types) is removed. Then rename toPartialGqlInfo, graphqlFields, and toGraphQL
function createFolder(folder: string) {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`);
        fs.mkdirSync(folder);
    }
}

function deleteFile(file: string) {
    if (fs.existsSync(file)) {
        console.info(`Deleting file: ${file}`);
        fs.unlinkSync(file);
    }
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
    const restFolder = "../server/src/endpoints/generated";
    createFolder(sharedOutputFolder);
    createFolder(restFolder);

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
            const data = await endpointGroup[endpointName] as object;
            console.log('got first endpoint data', endpointName, JSON.stringify(data));
            process.exit();
            // // Add the fragments to the total fragment object
            // for (const [name, fragment] of fragments) {
            //     allFragments[name] = fragment;
            // }
            // // Calculate imports, startig with the gql import
            // let importsString = "import gql from \"graphql-tag\";";
            // // Add import for each fragment
            // for (const [fragmentName] of fragments) {
            //     importsString += `\nimport { ${fragmentName} } from "../fragments/${fragmentName}";`;
            // }
            // // Create full endpoint string and file name
            // const endpointString = `export const ${objectType}${uppercaseFirstLetter(endpointName)} = gql\`${tag}\`;\n\n`;
            // const outputFile = `${objectType}_${endpointName}`;
            // // Store in total endpoint object
            // allEndpoints[outputFile] = `${importsString}\n\n${endpointString}`;
        }
    }
    process.exit();

    // Find GraphQL fragments
    // Note: These do not use gql tags because they are always used in other gql tags. 
    // If we made these gql tags, we'd have to import nested fragments, which might 
    // cause duplicate imports.
    for (const [name, fragment] of Object.entries(allFragments)) {
        allFragments[name] = `export const ${name} = \`${fragment}\`;\n`;
    }

    // Convert endpoint tags to PartialGraphQLInfo objects, for use in REST endpoints
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

    // Generate endpoint/method pairs for each endpoint
    console.info("Generating endpoint/method pairs...");
    const pairsSrcFolder = "../server/src/endpoints/rest";
    const pairFilePath = `${sharedOutputFolder}/pairs.ts`;

    // Find the name of every file in the pairs folder
    const excludedFiles = ["base", "index", "payment", "root", "types"];
    const pairFiles = fs.readdirSync(pairsSrcFolder).map(file => file.replace(".ts", "")).filter(file => !excludedFiles.includes(file));

    // Create backup of pairs file
    if (fs.existsSync(pairFilePath)) {
        fs.copyFileSync(pairFilePath, `${pairFilePath}.bak`);
    }
    // Clear the pairs file
    fs.writeFileSync(pairFilePath, "");

    for (const pairFile of pairFiles) {
        try {
            // Import rest file
            await import(`../../../${pairsSrcFolder}/${pairFile}.ts`).then(m => {
                // Find router
                const router = m[`${pairFile.charAt(0).toUpperCase() + pairFile.slice(1)}Rest`];
                if (!router) {
                    console.error(`No endpoint router found in ${pairFile}.ts. If this is not an error, consider adding to to the excludedFiles array.`);
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
                                tag: pairFile,
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
            console.error(`Error processing REST file ${pairFile}.ts: ${error}`);
            // Restore pairs file from backup
            if (fs.existsSync(`${pairFilePath}.bak`)) {
                fs.copyFileSync(`${pairFilePath}.bak`, pairFilePath);
            }
        } finally {
            // Delete pairs file backup
            deleteFile(`${pairFilePath}.bak`);
        }
    }

    // Create index.ts for generated folder
    const indexArray = ["graphqlTypes"];
    // indexArray.push("rest"); // Stored in the server instead, so not referenced in shared index.ts
    indexArray.push("pairs");
    const indexContent = indexArray.map(name => `export * from "./${name}";`).join("\n");
    fs.writeFileSync(`${sharedOutputFolder}/index.ts`, indexContent);

    console.info("Finished generating graphql-tag strings for endpoints🚀");
}

main().catch(console.error).finally(() => process.exit());
