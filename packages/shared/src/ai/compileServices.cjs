/* eslint-disable @typescript-eslint/no-var-requires */
const fs = require("fs");
const path = require("path");

const IMPORT_RELATIVE_PATH = "../../dist/ai";
const IMPORT_ABSOLUTE_PATH = path.join(__dirname, IMPORT_RELATIVE_PATH);
const IMPORT_FILE_NAME = "services";
const IMPORT_FILE_EXTENSION = ".js";
const OUTPUT_RELATIVE_PATH = "../../dist/ai/configs/services";
const OUTPUT_ABSOLUTE_PATH = path.join(__dirname, OUTPUT_RELATIVE_PATH);
const OUTPUT_FILE_NAME = "services";
const OUTPUT_FILE_EXTENSION = ".json";
const MINIFY = true;

/**
 * Converts AI service configuration into a JSON file that can be 
 * safely imported by the client.
 */
async function compileToJson() {
    // Import the file 
    const { aiServicesInfo } = await import(path.join(IMPORT_ABSOLUTE_PATH, `${IMPORT_FILE_NAME}${IMPORT_FILE_EXTENSION}`).replace(/\\/g, "/"));

    // Stringify the results
    const stringified = MINIFY ? JSON.stringify(aiServicesInfo) : JSON.stringify(aiServicesInfo, null, 2);

    // Write the results to a JSON file. Make sure any missing directories are created.
    if (!fs.existsSync(OUTPUT_ABSOLUTE_PATH)) {
        fs.mkdirSync(OUTPUT_ABSOLUTE_PATH, { recursive: true });
    }
    fs.writeFileSync(path.join(OUTPUT_ABSOLUTE_PATH, `${OUTPUT_FILE_NAME}${OUTPUT_FILE_EXTENSION}`), stringified);
}

compileToJson().catch(console.error);
