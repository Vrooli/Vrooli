/**
 * This script converts our API information to an OpenAPI schema.
 * This is useful for: 
 * - Understanding the API
 * - Parsing additional information (such as upserting standards for main Vrooli objects)
 * - Supporting ChatGPT plugin functionality.
 * 
 * It takes advantage of:
 * - The auto-generated GraphQL types
 * - Our generated endpoint pairs, used by the UI to make requests to the REST API
 * 
 * To run this script, run `ts-node --esm --experimental-specifier-resolution node ./src/tools/gqlToJson.ts` from the `packages/shared` directory.
 */
import * as fs from "fs";
import * as path from "path";
import { getOpenApiWriter, getTypeScriptReader, makeConverter } from "typeconv";
import { fileURLToPath } from "url";

type RestPair = {
    endpoint: string;
    method: "GET" | "POST" | "PUT" | "DELETE";
    name: string;
    tag?: string;
}

// File locations
const GQL_TYPES_FILE = "../api/generated/graphqlTypes.ts";
const REST_PAIRS_FILE = "../api/generated/pairs.ts";

// Get the directory name of the current module
const dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Converts plaintext of REST endpoint pairs file to an array of RestPair objects.
 */
const parseEndpoints = (): RestPair[] => {
    // Initialize the REST endpoint pairs array
    const restPairs: RestPair[] = [];

    // Read the file containing the REST endpoint pairs
    const pathsToRestPairsFile = path.resolve(dirname, REST_PAIRS_FILE);
    if (!fs.existsSync(pathsToRestPairsFile)) {
        console.error("No REST endpoint pairs file found, so schema options will not be generated.");
        return restPairs;
    }
    const restData = fs.readFileSync(pathsToRestPairsFile, "utf8");

    // Each type is separated by a blank line, so we can split on that
    const restBlocks = restData.split("\n\n");

    // Define regular expressions to match data in each block
    const nameRegex = /export const (.*?) =/s;
    const endpointRegex = /endpoint: "(.*?)"/s;
    const methodRegex = /method: "(.*?)"/s;
    const tagRegex = /tag: "(.*?)"/s;

    for (const block of restBlocks) {
        const nameMatch = block.match(nameRegex);
        const endpointMatch = block.match(endpointRegex);
        const methodMatch = block.match(methodRegex);
        const tagMatch = block.match(tagRegex);

        if (nameMatch && endpointMatch && methodMatch) {
            const restPair: RestPair = {
                name: nameMatch[1],
                endpoint: endpointMatch[1],
                method: methodMatch[1] as "GET" | "POST" | "PUT" | "DELETE",
                tag: tagMatch ? tagMatch[1] : undefined,
            };
            restPairs.push(restPair);
        }
    }

    console.log(restPairs);
    return restPairs;
};

/**
 * Adds an array of RestPair objects as operations in an OpenAPI schema.
 */
const addOperationsToSchema = (schema: Record<string, any>, restPairs: RestPair[]): void => {
    for (const pair of restPairs) {
        // Convert the path to OpenAPI format (colon-prefixed path parameters to curly braces)
        const path = pair.endpoint.replace(/:(\w+)/g, "{$1}");
        // Ensure that the paths object exists in the schema
        schema.paths = schema.paths || {};
        // Ensure that the path object exists in the schema
        schema.paths[path] = schema.paths[path] || {};
        // Add the operation
        schema.paths[path][pair.method.toLowerCase()] = {
            operationId: pair.name,
            tags: pair.tag ? [pair.tag] : undefined,
            // This line assumes that all endpoints do not require parameters. 
            // If this is not the case, you would need to add logic to define them.
            parameters: [],
            responses: {
                // A default response. Real responses should be defined based on the actual API.
                default: {
                    description: "Default response. Replace this with real responses.",
                },
            },
        };
    }
};

/**
 * Shapes plaintext of GraphQL types file for conversion to OpenAPI schema.
 */
const parseGqlTypes = (): string => {
    // Read the file containing the GraphQL types
    const pathsToGqlTypesFile = path.resolve(dirname, GQL_TYPES_FILE);
    let gqlData = fs.readFileSync(pathsToGqlTypesFile, "utf8");

    // Split into blocks of types
    let blocks = gqlData.split("\n\n");

    // Remove the first block, which contains imports, Maybe, InputMaybe, and other types which are not used by the REST API
    blocks.shift();

    // Filter out other GraphQL-specific types from the rest of the blocks:
    // - Anything with `Resolver` in the name
    // - Maybe
    // - InputMaybe
    // - Exact
    // - MakeOptional
    // - MakeMaybe
    // - Omit
    // - RequireFields
    // - Scalars
    // - Mutation and Mutation...Args
    // - Query and Query...Args
    // Most of these were probably included in the first block, but depending on spacing maybe not
    blocks = blocks.filter(block => {
        // Get the first line of code (excluding comments)
        const firstLineOfCode = block.split("\n").find(line => !line.trim().startsWith("//") && !line.trim().startsWith("/*") && line.trim().length > 0);
        return !firstLineOfCode?.match(/export type .*Resolver.*/) &&
            !firstLineOfCode?.startsWith("export type Maybe<T> =") &&
            !firstLineOfCode?.startsWith("export type InputMaybe<T> = ") &&
            !firstLineOfCode?.startsWith("export type Exact<T extends ") &&
            !firstLineOfCode?.startsWith("export type MakeOptional<T, K extends ") &&
            !firstLineOfCode?.startsWith("export type MakeMaybe<T, K extends ") &&
            !firstLineOfCode?.startsWith("export type Omit<T, K extends ") &&
            !firstLineOfCode?.startsWith("export type RequireFields<T, K extends ") &&
            !firstLineOfCode?.startsWith("export type Scalars = ") &&
            !firstLineOfCode?.match(/export type Mutation[A-Za-z]* = /) &&
            !firstLineOfCode?.match(/export type Query[A-Za-z]* = /);
    });

    // Define a Map to store the enums and their corresponding string unions
    const enumMap = new Map<string, string>();

    // Convert TypeScript enums to string unions 
    // NOTE: Does not modify gqlData - just stores the enum and its corresponding union in the map
    gqlData.replace(/export enum (\w+) \{([^}]+)\}/g, (match, enumName, enumBody) => {
        // Split the body of the enum into individual members
        const enumMembers = enumBody.split(",");

        // Map each member to its value (the part after the equals sign)
        const enumValues = enumMembers.map(member => {
            const parts = member.trim().split("=");
            return parts[1]?.trim() || parts[0]?.trim();
        });

        // Join the values into a union
        const union = enumValues.join(" | ");

        // Store the enum and its corresponding union in the map
        enumMap.set(enumName, union);

        // Return an empty string to remove the enum block
        return "";
    });

    // Remove enum blocks from the array of blocks
    blocks = blocks.filter(block => !block.startsWith("export enum "));

    // Replace enum usages with their corresponding unions
    blocks = blocks.map(block => {
        let replacedBlock = block;
        for (const [enumName, union] of enumMap.entries()) {
            const enumRegex = new RegExp(`\\b${enumName}\\b`, "g");
            replacedBlock = replacedBlock.replace(enumRegex, union);
        }
        return replacedBlock;
    });

    // Join the blocks back together
    gqlData = blocks.join("\n\n");

    // Replace GraphQL's custom scalar types with TypeScript's built-in types
    gqlData = gqlData.replace(/Scalars\['[A-Za-z]+'\]/g, (match) => {
        // Includes all built-in types, and any additional custom types (typically defined in `root` GraphQL typeDef)
        switch (match) {
            case "Scalars['Boolean']":
                return "boolean";
            case "Scalars['Date']":
                return "string";
            case "Scalars['Float']":
                return "number";
            case "Scalars['ID']":
                return "string";
            case "Scalars['Int']":
                return "number";
            case "Scalars['String']":
                return "string";
            case "Scalars['Upload']":
                return "unknown";
            default:
                throw new Error(`Unknown scalar type: ${match}`);
        }
    });

    // Replace GraphQL's `InputMaybe<T>` and `Maybe<T>` with TypeScript's `T | null`
    gqlData = gqlData.replace(/InputMaybe<Array<([^>]+)>>/g, "Array<$1> | null | undefined");
    gqlData = gqlData.replace(/Maybe<Array<([^>]+)>>/g, "Array<$1> | null | undefined");
    gqlData = gqlData.replace(/InputMaybe<([^>]+)>/g, "$1 | null | undefined");
    gqlData = gqlData.replace(/Maybe<([^>]+)>/g, "$1 | null | undefined");

    // Return shaped GraphQL types
    return gqlData;
};

const main = async () => {

    // Create the reader and writer
    const reader = getTypeScriptReader();
    const writer = getOpenApiWriter({ format: "json", title: "Vrooli", version: "1.9.4" });

    // Create the converter
    const { convert } = makeConverter(reader, writer);

    // Get endpoint pairs
    const restPairs = parseEndpoints();

    // Get GraphQL types
    const gqlData = parseGqlTypes();

    // Convert the GraphQL types types to an OpenAPI schema
    let { data } = await convert({ data: gqlData });

    // Parse the JSON
    const openApiSchema = JSON.parse(data);

    // Add additional information about our company and API
    openApiSchema.info.description = "This is the REST API for Vrooli. Find out more about Vrooli at https://vrooli.com/about";
    openApiSchema.info.termsOfService = "https://vrooli.com/terms";
    openApiSchema.info.contact = {
        name: "Vrooli API Support",
        email: "support@vrooli.com",
    };
    openApiSchema.info.license = {
        name: "GPL-3.0",
        url: "https://opensource.org/licenses/GPL-3.0",
    };

    // Add the REST endpoints as operations (paths)
    addOperationsToSchema(openApiSchema, restPairs);

    // Stringify the JSON
    data = JSON.stringify(openApiSchema, null, 2);

    // Write the OpenAPI schema to a file
    fs.writeFile(path.resolve(dirname, "../../../docs/docs/assets/openapi.json"), data, (err) => {
        if (err) throw err;
        console.log("The OpenAPI schema has been saved!");
    });
};

main().catch(console.error);
