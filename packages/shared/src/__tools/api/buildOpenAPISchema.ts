/**
 * This script converts our API information to an OpenAPI schema.
 * This is useful for: 
 * - Understanding the API
 * - Parsing additional information (such as upserting standards for main Vrooli objects)
 * - Supporting ChatGPT plugin functionality.
 * 
 * It takes advantage of:
 * - Our generated endpoint pairs, used by the UI to make requests to the REST API
 * - Our server endpoints logic, which defines the output of each REST API endpoint
 * - Our TypeScript types, which define the structure of our API objects
 */
import fs from "fs";
import path from "path";
import * as TJS from "typescript-json-schema";
import { fileURLToPath } from "url";
import * as pairs from "../../api/pairs";
import { HttpStatus } from "../../consts/api";
import { uppercaseFirstLetter } from "../../utils/casing";

type Method = "GET" | "POST" | "PUT" | "DELETE";
type RestPair = {
    endpoint: string;
    method: Method;
    objectType: string;
    operationId: string;
    operationType: string;
}

type SchemaInfo = {
    title: string;
    version: string;
    description: string;
    termsOfService: string;
    contact: {
        name: string;
        email: string;
    };
    license: {
        name: string;
        url: string;
    };
}
type ReferenceObject = {
    $ref: string;
}
type SchemaContent = {
    [key: string]: {
        schema: ReferenceObject;
    };
}
type ContentWithDescription = {
    description: string;
    content: SchemaContent;
}
type SchemaPathEndpointResponse = ContentWithDescription | ReferenceObject
type SchemaPathEndpoint = {
    summary: string | undefined;
    description: string | undefined;
    operationId: string;
    tags: string[];
    parameters: any[];
    requestBody?: {
        required: boolean;
        content: SchemaContent;
    };
    responses: Record<string, SchemaPathEndpointResponse>;
}
type SchemaComponentsResponses = {
    [key: string]: {
        description: string;
        content: {
            [key: string]: {
                schema: {
                    $ref: string;
                }
            };
        };
    };
}
type SchemaComponentsSchemas = {
    [key: string]: {
        type: string;
        properties: Record<string, any>;
        required: string[];
    };
}
type SchemaComponents = {
    responses: SchemaComponentsResponses;
    schemas: SchemaComponentsSchemas;
}
type OpenApiSchema = {
    openapi: string;
    info: SchemaInfo;
    paths: Record<string, Record<string, SchemaPathEndpoint>>;
    components: SchemaComponents;
}

/** 
 * A structure to hold discovered Endpoint definitions by `objectType` and operation key.
 * Example:
 *   endpointsByType["Bookmark"] = {
 *       findOne:  { inputType: "FindByIdInput", outputType: "Bookmark" },
 *       findMany: { inputType: "BookmarkSearchInput", outputType: "BookmarkSearchResult" },
 *       ...
 *   }
 */
type EndpointDefinition = { inputType: string | null; outputType: string };
type EndpointsMap = {
    [objectType: string]: {
        [operationKey: string]: EndpointDefinition;
    };
};

// File locations
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageJsonPath = path.resolve(__dirname, "../../../package.json");
const endpointsLogicPath = path.resolve(__dirname, "../../../../server/src/endpoints/logic");
const outputPath = path.resolve(__dirname, "../../../../docs/docs/assets/openapi.json");
const apiTypesPath = path.resolve(__dirname, "../../api/types.ts");

function getArticle(name: string) {
    if (name.length === 0) return "a";
    const firstLetter = name[0].toLowerCase();
    return "aeiou".includes(firstLetter) ? "an" : "a";
}
function findOneInfo(objectType: string) {
    return {
        summary: `Retrieve a single ${objectType}`,
        description: `Returns a single ${objectType} that matches a provided ID.`,
    };
}
function findManyInfo(objectType: string) {
    return {
        summary: `Retrieve many ${objectType}`,
        description: `Returns multiple ${objectType}s with cursor, optionally filtered.`,
    };
}
function createOneInfo(objectType: string) {
    return {
        summary: `Create a new ${objectType}`,
        description: `Creates a new ${objectType} object with the provided data.`,
    };
}
function createManyInfo(objectType: string) {
    return {
        summary: `Create multiple ${objectType}`,
        description: `Creates multiple ${objectType} objects with the provided data.`,
    };
}
function updateOneInfo(objectType: string) {
    const article = getArticle(objectType);
    return {
        summary: `Update an existing ${objectType}`,
        description: `Updates ${article} ${objectType} with the provided data.`,
    };
}
function updateManyInfo(objectType: string) {
    return {
        summary: `Update multiple ${objectType}`,
        description: `Updates multiple ${objectType} objects with the provided data.`,
    };
}
function deleteOneInfo(objectType: string) {
    const article = getArticle(objectType);
    return {
        summary: `Delete ${article} ${objectType}`,
        description: `Deletes ${article} ${objectType} with the provided ID.`,
    };
}

/**
 * A map of endpoint summaries and descriptions by `objectType` and operation key. 
 * If not provided, the default values are used.
 */
const ENDPOINT_INFO: Record<
    string,
    Record<string, { summary: string; description: string }>
> = {
    Actions: {
        copy: {
            summary: "Copy an object",
            description: "Create a copy of the provided object",
        },
        deleteOne: {
            summary: "Delete an object",
            description: "Delete an object with the provided ID",
        },
        deleteMany: {
            summary: "Delete multiple objects",
            description: "Delete multiple objects with the provided IDs",
        },
        deleteAll: {
            summary: "Delete objects of type",
            description: "Delete all objects of the provided type",
        },
        deleteAccount: {
            summary: "Delete account",
            description: "Delete your account",
        },
    },
    Auth: {
        emailLogIn: {
            summary: "Log in with email",
            description: "Log in with email and password",
        },
        emailSignUp: {
            summary: "Sign up with email",
            description: "Sign up with email and password",
        },
        emailRequestPasswordChange: {
            summary: "Request password change",
            description: "Request a password change",
        },
        emailResetPassword: {
            summary: "Reset password",
            description: "Reset your password",
        },
        guestLogIn: {
            summary: "Log in as guest",
            description: "Log in as a guest user",
        },
        logout: {
            summary: "Log out",
            description: "Log out the current user",
        },
        logoutAll: {
            summary: "Log out all",
            description: "Log out all sessions",
        },
        validateSession: {
            summary: "Validate session",
            description: "Validate the current session",
        },
        switchCurrentAccount: {
            summary: "Switch current account",
            description: "Switch the current account",
        },
        walletInit: {
            summary: "Initialize wallet handshake",
            description: "Initialize crypto wallet handshake for authentication",
        },
        walletComplete: {
            summary: "Complete wallet handshake",
            description: "Complete crypto wallet handshake for authentication",
        },
    },
};

/**
 * Returns the best-fit summary & description for the given objectType and operationType.
 * 1) If there's a custom entry in `ENDPOINT_INFO[objectType][operationType]`, use it.
 * 2) Otherwise, if the operationType matches one of the known default functions,
 *    return that.
 * 3) Otherwise, we return a fallback with an error message.
 */
function getEndpointInfo(objectType: string, operationType: string): { summary: string | undefined; description: string | undefined } {
    // 1) Check the map
    if (ENDPOINT_INFO[objectType]?.[operationType]) {
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        return ENDPOINT_INFO[objectType]![operationType]!;
    }

    // 2) Check known operation types
    if (operationType === "findOne") return findOneInfo(objectType);
    if (operationType === "findMany") return findManyInfo(objectType);
    if (operationType === "createOne") return createOneInfo(objectType);
    if (operationType === "createMany") return createManyInfo(objectType);
    if (operationType === "updateOne") return updateOneInfo(objectType);
    if (operationType === "updateMany") return updateManyInfo(objectType);
    if (operationType === "deleteOne") return deleteOneInfo(objectType);

    console.warn(`Warning: No endpoint info found for ${objectType}.${operationType}. Cannot generate summary/description.`);
    return { summary: undefined, description: undefined };
}

function referenceResponseId(responseType: string): string {
    return `#/components/responses/${responseType}`;
}
function referenceSchemaId(objectType: string): string {
    return `#/components/schemas/${objectType}`;
}

function referenceResponse(responseType: string): ReferenceObject {
    return {
        $ref: referenceResponseId(responseType),
    };
}
function referenceSchema(objectType: string): ReferenceObject {
    return {
        $ref: referenceSchemaId(objectType),
    };
}

function refenceSchemaContent(objectType: string, contentType = "application/json"): { content: SchemaContent } {
    return {
        content: {
            [contentType]: {
                schema: referenceSchema(objectType),
            },
        },
    };
}

const ERRORS = {
    [HttpStatus.BadRequest]: {
        key: "BadRequest",
        description: "Bad Request",
    },
    [HttpStatus.Forbidden]: {
        key: "UnsafeOrigin",
        description: "Unsafe origin",
    },
    [HttpStatus.InternalServerError]: {
        key: "InternalServerError",
        description: "Internal Server Error",
    },
    [HttpStatus.Unauthorized]: {
        key: "NotLoggedIn",
        description: "Not logged in",
    },
};

const initialComponents = {
    responses: {
        // Common responses for all endpoints (typically authentication and unexpected errors)
        [ERRORS[HttpStatus.BadRequest].key]: {
            description: ERRORS[HttpStatus.BadRequest].description,
            ...refenceSchemaContent("Error"),
        },
        [ERRORS[HttpStatus.Forbidden].key]: {
            description: ERRORS[HttpStatus.Forbidden].description,
            ...refenceSchemaContent("Error"),
        },
        [ERRORS[HttpStatus.InternalServerError].key]: {
            description: ERRORS[HttpStatus.InternalServerError].description,
            ...refenceSchemaContent("Error"),
        },
    },
    schemas: {
        // Common types for multiple endpoints
        Error: {
            type: "object",
            properties: {
                code: {
                    type: "integer",
                },
                message: {
                    type: "string",
                },
                details: {
                    type: "string",
                },
            },
            required: ["code", "message"],
        },
    },
};

/**
 * Converts plaintext of REST endpoint pairs file to an array of RestPair objects.
 */
function parseEndpoints(): RestPair[] {
    // Initialize the REST endpoint pairs array
    const restPairs: RestPair[] = [];

    // Loop through each item in the pairs object
    Object.entries(pairs).forEach(([endpointGroupKey, endpointGroupValue]) => {
        // If it doesn't start with "endpoints" or isn't an object, skip it
        if (!endpointGroupKey.startsWith("endpoints") || typeof endpointGroupValue !== "object") {
            return;
        }
        // Loop through each item in the object
        Object.entries(endpointGroupValue).forEach(([endpointKey, endpointValue]) => {
            // If object does not match the expected shape, skip it
            if (
                !Object.prototype.hasOwnProperty.call(endpointValue, "endpoint") ||
                !Object.prototype.hasOwnProperty.call(endpointValue, "method")
            ) {
                return;
            }
            // Add the endpoint to the array
            restPairs.push({
                endpoint: (endpointValue as unknown as { endpoint: string }).endpoint,
                method: (endpointValue as unknown as { method: Method }).method,
                objectType: endpointGroupKey.substring("endpoints".length),
                operationId: `${endpointGroupKey}${uppercaseFirstLetter(endpointKey)}`,
                operationType: endpointKey,
            });
        });
    });

    return restPairs;
}

/**
 * Finds the "author" and "version" fields in the package.json file.
 */
function getPackageJsonInfo(): { author: string; version: string } {
    if (!fs.existsSync(packageJsonPath)) {
        console.error("package.json file not found. OpenAPI schema will be missing some project information.");
        return { author: "", version: "" };
    }

    // Read the package.json file
    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
    // Get the author and version fields
    const author = packageJson.author;
    const version = packageJson.version;
    return { author, version };
}

/**
 * Parse each endpoints logic file in `endpointsLogicPath` to collect
 * the input/output type for each endpoint.
 *
 * We look for a pattern like:
 *   export type EndpointsBookmark = {
 *       findOne: ApiEndpoint<FindByIdInput, Bookmark>;
 *       ...
 *   }
 *
 * Ideally we'd probably do a more robust parse (AST),
 * but here we do a rough (regex-based) search.
 */
function parseEndpointLogicFiles(): EndpointsMap {
    const endpointsByType: EndpointsMap = {};

    // Read all *.ts files in endpointsLogicPath
    const files = fs.readdirSync(endpointsLogicPath).filter((f) => f.endsWith(".ts"));

    for (const file of files) {
        const fullPath = path.join(endpointsLogicPath, file);
        const fileContent = fs.readFileSync(fullPath, "utf-8");

        // Heuristic: the file might contain something like:
        //    export type EndpointsBookmark = {
        //        findOne: ApiEndpoint<FindByIdInput, Bookmark>;
        //        findMany: ApiEndpoint<BookmarkSearchInput, BookmarkSearchResult>;
        //        ...
        //    }
        //
        // We'll:
        //  1. Find the string "export type EndpointsXXX = { ... }"
        //  2. Capture everything inside the curly braces
        //  3. For each line, parse out the key and the generic <inputType, outputType>
        //
        // We'll also try to guess the objectType from "EndpointsBookmark" => "Bookmark"
        const endpointsTypeRegex = /export\s+type\s+Endpoints([A-Z]\w+)\s*=\s*\{([^}]+)\}/gm;
        let match;
        while ((match = endpointsTypeRegex.exec(fileContent)) !== null) {
            // match[1] => the part after "Endpoints", e.g. "Bookmark"
            // match[2] => the full text inside the curly braces
            const objectType = match[1]; // e.g. "Bookmark"
            const bodyContent = match[2];

            // For lines like:  findOne: ApiEndpoint<FindByIdInput, Bookmark>;
            const lineRegex = /(\w+)\s*:\s*ApiEndpoint<([^>]+)>/;

            // Loop through each line in the bodyContent
            for (const line of bodyContent.split("\n")) {
                const trimmed = line.trim();
                const lineMatch = trimmed.match(lineRegex);
                if (!lineMatch) continue;

                const operationKey = lineMatch[1]; // e.g. "findOne", "guestLogIn"
                const insideGenerics = lineMatch[2].trim(); // e.g. "FindByIdInput, Bookmark"

                // Split by comma
                const parts = insideGenerics.split(",");
                if (parts.length < 2) continue; // not well-formed; skip

                // Need exactly two parts: inputType, outputType.
                // If there are extra, combine them into the inputType.
                const inputType = parts.slice(0, -1).join(",").trim();
                const outputType = parts[parts.length - 1].trim();

                const endpointDefinition: EndpointDefinition = {
                    // Ignore inputs that don't end with "Input" (e.g. "never")
                    inputType: inputType.endsWith("Input") ? inputType : null,
                    outputType,
                };
                endpointsByType[objectType] = {
                    ...endpointsByType[objectType],
                    [operationKey]: endpointDefinition,
                };
            }
        }
    }

    return endpointsByType;
}

/**
 * Populates the OpenAPI schema "paths" object with data parsed from the REST endpoint pairs.
 */
function setSchemaPaths(
    schema: OpenApiSchema,
    restPairs: RestPair[],
    endpointsByType: EndpointsMap,
): void {
    const result: Record<string, Record<string, SchemaPathEndpoint>> = {};
    for (const pair of restPairs) {
        // Convert the path to OpenAPI format (colon-prefixed path parameters to curly braces)
        const path = pair.endpoint.replace(/:(\w+)/g, "{$1}");
        const endpoint = pair.method.toLowerCase();

        // Find the input and output types for this endpoint, if available
        const endpointDefinition = endpointsByType[pair.objectType]?.[pair.operationType];
        if (!endpointDefinition) {
            console.error(`No endpoint definition found for ${pair.objectType}.${pair.operationType}`);
            continue;
        }
        const { inputType, outputType } = endpointDefinition;

        // Construct path parameters array if we detect any path variable :xxx
        const pathParamsMatches = pair.endpoint.match(/:(\w+)/g) || [];
        const parameters = pathParamsMatches.map((p) => {
            // p would look like ":id"
            const paramName = p.slice(1); // remove leading colon
            return {
                name: paramName,
                in: "path",
                required: true,
                schema: {
                    type: "string", // or "number", or parse further if you wish
                },
            };
        });

        // The responses may change depending on the operation type, but 
        // the basic success and authentication responses are the same
        const responses: Record<string, SchemaPathEndpointResponse> = {
            [HttpStatus.Ok.toString()]: {
                description: "Successful response",
                ...refenceSchemaContent(outputType),
            },
            [HttpStatus.BadRequest.toString()]: {
                ...referenceResponse(ERRORS[HttpStatus.BadRequest].key),
            },
            [HttpStatus.Forbidden.toString()]: {
                ...referenceResponse(ERRORS[HttpStatus.Forbidden].key),
            },
            [HttpStatus.InternalServerError.toString()]: {
                ...referenceResponse(ERRORS[HttpStatus.InternalServerError].key),
            },
        };
        // Add more responses as needed
        switch (pair.operationType) {
            case "findOne": {
                responses[HttpStatus.NotFound.toString()] = {
                    description: `${pair.objectType} not found`,
                    ...refenceSchemaContent("Error"),
                };
                //...
                break;
            }
            case "findMany": {
                //...
                break;
            }
            case "createOne": {
                responses[HttpStatus.Unauthorized.toString()] = {
                    description: ERRORS[HttpStatus.Unauthorized].description,
                    ...refenceSchemaContent("Error"),
                };
                //...
                break;
            }
            case "updateOne": {
                responses[HttpStatus.NotFound.toString()] = {
                    description: `${pair.objectType} not found`,
                    ...refenceSchemaContent("Error"),
                };
                responses[HttpStatus.Unauthorized.toString()] = {
                    description: ERRORS[HttpStatus.Unauthorized].description,
                    ...refenceSchemaContent("Error"),
                };
                //...
                break;
            }
            // Add more cases as needed
        }
        const { summary, description } = getEndpointInfo(pair.objectType, pair.operationType);
        // Build the endpoint path object
        const pathObject: SchemaPathEndpoint = {
            summary,
            description,
            operationId: pair.operationId,
            tags: [pair.objectType],
            parameters,
            responses,
        };
        // If we expect a request body (POST, PUT, PATCH?), add a requestBody section
        if (["post", "put", "patch"].includes(pair.method.toLowerCase()) && inputType) {
            // For example, reference the inputType as the body schema
            pathObject.requestBody = {
                required: true,
                ...refenceSchemaContent(inputType),
            };
        }
        // Add the path object to the result
        result[path] = { ...result[path], [endpoint]: pathObject };
    }
    // Add the result to the schema
    schema.paths = result;
}

/**
 * Build a JSON Schema generator, using `typescript-json-schema`.
 */
function buildJsonSchemaGenerator(tsFilePath: string) {
    // Adjust these settings and compilerOptions as needed:
    const settings: TJS.PartialArgs = {
        required: true,
        // You could turn on "topRef" to only generate for a single root type, etc.
        // See https://github.com/YousefED/typescript-json-schema#usage
        // topRef: false,
    };

    const compilerOptions: TJS.CompilerOptions = {
        // e.g. "strictNullChecks": true, "esModuleInterop": true, etc. Adjust to match your project’s tsconfig
        strictNullChecks: true,
    };

    // Create a program from the file(s)
    const program = TJS.getProgramFromFiles([tsFilePath], compilerOptions);
    // Build the generator
    const generator = TJS.buildGenerator(program, settings);
    if (!generator) {
        throw new Error("Failed to build typescript-json-schema generator.");
    }
    return generator;
}

/**
* Parse your `api/types.ts` using `typescript-json-schema` and return a record of
* { [typeName]: JSONSchema } for all user-defined types.
*/
async function parseApiTypes(): Promise<Record<string, any>> {
    if (!fs.existsSync(apiTypesPath)) {
        console.warn(
            `Warning: Could not find ${apiTypesPath}. No type definitions will be parsed.`,
        );
        return {};
    }

    const generator = buildJsonSchemaGenerator(apiTypesPath);
    // Get all user-defined symbols in the file
    const allSymbols = generator.getUserSymbols();

    // Or you can limit to specific types:
    // const allSymbols = ["MySpecificType", "AnotherType"];
    // but here we grab all user-defined symbols.

    const schemas: Record<string, any> = {};

    for (const symbol of allSymbols) {
        // This returns the JSON Schema for that symbol
        const schema = generator.getSchemaForSymbol(symbol);
        schemas[symbol] = schema;
    }

    return schemas;
}

/**
 * Populates the OpenAPI schema "components" object with data parsed from the REST endpoint logic files.
 */
function setSchemaComponents(
    schema: OpenApiSchema,
    endpointsByType: EndpointsMap,
): void {
    const result: SchemaComponents = { ...initialComponents };

    // Collect every inputType and outputType we encountered
    const allTypes = new Set<string>();
    Object.entries(endpointsByType).forEach(([objectType, endpointsMap]) => {
        Object.values(endpointsMap).forEach((def) => {
            if (def.inputType) {
                allTypes.add(def.inputType);
            }
            allTypes.add(def.outputType);
        });
    });

    // 2) Parse the main type file
    // const shapedApiTypes = parseApiTypes();
    // console.log("shapedApiTypes", shapedApiTypes);

    //TODO
    // Make sure each type is present in result.schemas
    // Here we just stub out "type: object" to let docs reference them. 
    // For an actual solution, you could do a TS -> OpenAPI conversion for each type, or a partial approach.
    for (const typeName of allTypes) {
        if (!result.schemas[typeName]) {
            result.schemas[typeName] = {
                type: "object",
                properties: {},
                required: [],
            };
        }
    }

    schema.components = result;
}

async function main() {
    console.info("Building OpenAPI schema. This may take a moment...");

    // Find project information using the package.json file
    const { author, version } = getPackageJsonInfo();

    // Get endpoint pairs
    const restPairs = parseEndpoints();

    // Parse each endpoints logic file to get input/output types
    const endpointsByType = parseEndpointLogicFiles();

    // Initialize the OpenAPI schema
    const openApiSchema: OpenApiSchema = {
        openapi: "3.0.0",
        info: {
            title: author,
            version,
            description: `This is the REST API for ${author}. Learn more at https://vrooli.com/about`,
            termsOfService: "https://vrooli.com/terms",
            contact: {
                name: `${author} API Support`,
                email: "support@vrooli.com",
            },
            license: {
                name: "AGPL-3.0",
                url: "https://choosealicense.com/licenses/agpl-3.0/",
            },
        },
        paths: {},
        components: {
            responses: {},
            schemas: {},
        },
    };

    // Add the REST endpoints as operations (paths)
    setSchemaPaths(openApiSchema, restPairs, endpointsByType);
    // Add responses and schemas to the components object
    setSchemaComponents(openApiSchema, endpointsByType);

    // Write the OpenAPI schema to a file
    // eslint-disable-next-line no-magic-numbers
    fs.writeFile(outputPath, JSON.stringify(openApiSchema, null, 4), (err) => {
        if (err) throw err;
        console.info("The OpenAPI schema has been saved!🥳");
    });
}

main().catch(console.error);
