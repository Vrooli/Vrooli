/* eslint-disable @typescript-eslint/no-var-requires */
const fs = require("fs");
const path = require("path");
const { pathToFileURL } = require("url");

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
 * Attempts to load a module using require. If that fails (e.g. if the module is ESM-only),
 * falls back to dynamic import. (Dynamic import returns a Promise so this function is async.)
 * 
 * @param {string} modulePath The path to the module to load.
 * @returns {Promise<*>} The module that was loaded.
 */
async function loadModule(modulePath) {
    try {
        // Try the standard CommonJS require first.
        return require(modulePath);
    } catch (err) {
        // If require fails (for example, if the module is ESM-only), fall back to dynamic import.
        // Convert the module path to a file URL.
        const fileUrl = pathToFileURL(modulePath).href;
        const imported = await import(fileUrl);
        // If the module was exported with a default, return that.
        return imported.default || imported;
    }
}

/**
 * Converts AI service configuration into a JSON file that can be 
 * safely imported by the client.
 */
async function compileToJson() {
    // Import the file using require instead of import
    const modulePath = path.join(IMPORT_ABSOLUTE_PATH, `${IMPORT_FILE_NAME}${IMPORT_FILE_EXTENSION}`);
    let aiServicesInfo;
    try {
        const servicesModule = await loadModule(modulePath);
        aiServicesInfo = servicesModule.aiServicesInfo;
        if (!aiServicesInfo) {
            console.error(`Error: aiServicesInfo is undefined in ${IMPORT_FILE_NAME}${IMPORT_FILE_EXTENSION}`);
            process.exit(1);
        }
    } catch (err) {
        console.error("Error importing services module:", err);
        process.exit(1);
    }

    // Stringify the results
    const stringified = MINIFY ? JSON.stringify(aiServicesInfo) : JSON.stringify(aiServicesInfo, null, 2);

    // Write the results to a JSON file. Make sure any missing directories are created.
    if (!fs.existsSync(OUTPUT_ABSOLUTE_PATH)) {
        fs.mkdirSync(OUTPUT_ABSOLUTE_PATH, { recursive: true });
    }
    fs.writeFileSync(path.join(OUTPUT_ABSOLUTE_PATH, `${OUTPUT_FILE_NAME}${OUTPUT_FILE_EXTENSION}`), stringified);
}

compileToJson().catch(console.error);
