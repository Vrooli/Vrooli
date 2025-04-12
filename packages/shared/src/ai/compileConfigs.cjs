/* eslint-disable @typescript-eslint/no-var-requires */
const fs = require("fs");
const path = require("path");
const { pathToFileURL } = require("url");

const CONFIG_RELATIVE_PATH = "../../dist/ai/configs/tasks";
const CONFIG_ABSOLUTE_PATH = path.join(__dirname, CONFIG_RELATIVE_PATH);
const FILE_EXTENSION = ".js";
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
 * Converts LLM configuration files (which are in TypeScript for code reuse and type-safety) 
 * into JSON files that can be safely imported by the client.
 */
async function compileToJson() {
    // Ensure CONFIG_ABSOLUTE_PATH exists
    if (!fs.existsSync(CONFIG_ABSOLUTE_PATH)) {
        console.error(`Config path does not exist: ${CONFIG_ABSOLUTE_PATH}`);
        process.exit(1);
    }

    // Find the names of all config files
    const configFiles = fs.readdirSync(CONFIG_ABSOLUTE_PATH).filter(file => file.endsWith(FILE_EXTENSION));

    // Initialize the results object, which will be a key of file names (without extension) and JSON objects
    const configResults = {};

    // Loop through each file
    for (const file of configFiles) {
        // Initialize the config object for this file
        const languageKey = file.replace(FILE_EXTENSION, "");
        configResults[languageKey] = {};

        // Import the config and builder
        const modulePath = path.join(CONFIG_ABSOLUTE_PATH, file);
        let imported;
        try {
            imported = await loadModule(modulePath);
        } catch (err) {
            console.error(`Error importing config file ${file}:`, err);
            continue;
        }
        const config = imported.config;
        // Make sure the config exists
        if (!config) {
            console.error(`Error: Config file ${file} does not export a config object.`);
            continue;
        }

        // Some config values are included directly without any processing
        const INCLUDE_DIRECTLY = ["__suggested_prefix", "__task_name_map"];
        for (const key of INCLUDE_DIRECTLY) {
            if (!configResults[languageKey]) configResults[languageKey] = {};
            configResults[languageKey][key] = config[key];
        }

        // Loop through each key in the config
        Object.keys(config).forEach(configKey => {
            // Only keys that start with a capital letter are task types. 
            if (configKey[0] < "A" || configKey[0] > "Z") return;
            try {
                // Call the function to get the task config
                const taskConfig = config[configKey]();
                // Add the task config to the results
                configResults[languageKey][configKey] = taskConfig;
            } catch (err) {
                console.error(`Error processing task config ${configKey} in ${file}:`, err);
            }
        });
    }

    // Loop through results and write to JSON files
    Object.keys(configResults).forEach(file => {
        const stringified = MINIFY ? JSON.stringify(configResults[file]) : JSON.stringify(configResults[file], null, 2);
        if (stringified === undefined) {
            console.error(`Stringified config for ${file} is undefined. Skipping write.`);
            return;
        }
        const outputPath = path.join(CONFIG_ABSOLUTE_PATH, `${file}.json`);
        fs.writeFileSync(outputPath, stringified);
    });
}

compileToJson().catch(console.error);
