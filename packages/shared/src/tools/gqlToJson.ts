// from packages/shared, run ts-node --esm --experimental-specifier-resolution node ./src/tools/gqlToJson.ts
import * as fs from "fs";
import * as path from "path";
import { getOpenApiWriter, getTypeScriptReader, makeConverter } from "typeconv";
import { fileURLToPath } from "url";

const main = async () => {
    // Get the directory name of the current module
    const dirname = path.dirname(fileURLToPath(import.meta.url));

    const pathToTsFile = path.resolve(dirname, "../api/generated/graphqlTypes.ts");

    // Read the TypeScript file
    let tsData = fs.readFileSync(pathToTsFile, "utf8");

    // Replace GraphQL's custom scalar types with TypeScript's built-in types
    tsData = tsData.replace(/Scalars\['[A-Za-z]+'\]/g, (match) => {
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
    tsData = tsData.replace(/InputMaybe<([^>]+)>/g, "$1 | null | undefined");
    tsData = tsData.replace(/Maybe<([^>]+)>/g, "$1 | null | undefined");

    //TODO for morning: generated types not correct. For example, ApiCreateInput.labelsConnect should be Array<string> | null, but is Array<string | null> instead. Also, nullable relations (not primitives) are not showing up in docs at all, meaning their generated type is invalid in some way. Also, enums are not showing up. Also, ask ChatGPT if we should be getting rid of ResolversTypes, as this might be only useful for GraphQL instead of REST
    // For testing purposes, print every line containing "createdBy" to the console
    tsData.split("\n").forEach((line) => {
        if (line.includes("createdBy")) {
            console.log(line);
        }
    });

    // Create the reader and writer
    const reader = getTypeScriptReader();
    const writer = getOpenApiWriter({ format: "json", title: "Vrooli", version: "1.9.4" });

    // Create the converter
    const { convert } = makeConverter(reader, writer);

    // Convert the TypeScript types to OpenAPI
    const { data } = await convert({ data: tsData });

    // Write the OpenAPI schema to a file
    fs.writeFile(path.resolve(dirname, "../../../docs/docs/assets/openapi.json"), data, (err) => {
        if (err) throw err;
        console.log("The OpenAPI schema has been saved!");
    });
};

main().catch(console.error);
