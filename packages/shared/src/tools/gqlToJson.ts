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
    const tsData = fs.readFileSync(pathToTsFile, "utf8");

    // Create the reader and writer
    const reader = getTypeScriptReader();
    const writer = getOpenApiWriter({ format: "json", title: "Vrooli", version: "1.9.4" });

    // Create the converter
    const { convert } = makeConverter(reader, writer);

    // Convert the TypeScript types to OpenAPI
    const { data } = await convert({ data: tsData });

    // Write the OpenAPI schema to a file
    fs.writeFile(path.resolve(dirname, "openapi.json"), data, (err) => {
        if (err) throw err;
        console.log("The OpenAPI schema has been saved!");
    });
};

main().catch(console.error);
