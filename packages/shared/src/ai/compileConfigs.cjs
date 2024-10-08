/* eslint-disable @typescript-eslint/no-var-requires */
const fs = require("fs");
const path = require("path");

const CONFIG_RELATIVE_PATH = "../../dist/ai/configs/tasks";
const CONFIG_ABSOLUTE_PATH = path.join(__dirname, CONFIG_RELATIVE_PATH);
const FILE_EXTENSION = ".js";
const MINIFY = true;

/**
 * Converts LLM configuration files (which are in TypeScript for code reuse and type-safety) 
 * into JSON files that can be safely imported by the client.
 */
async function compileToJson() {
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
        const { config } = await import(path.join(CONFIG_ABSOLUTE_PATH, file).replace(/\\/g, "/"));
        // Some config values are included directly without any processing
        const INCLUDE_DIRECTLY = ["__suggested_prefix", "__task_name_map"];
        for (const key of INCLUDE_DIRECTLY) {
            configResults[languageKey][key] = config[key];
        }
        // Loop through each key in the config
        Object.keys(config).forEach(configKey => {
            // Only keys that start with a capital letter are task types. 
            if (configKey[0] < "A" || configKey[0] > "Z") return;
            // Call the function to get the task config
            const taskConfig = config[configKey]();
            // Add the task config to the results
            configResults[languageKey][configKey] = taskConfig;
        });
    }

    // Loop through results
    Object.keys(configResults).forEach(file => {
        const stringified = MINIFY ? JSON.stringify(configResults[file]) : JSON.stringify(configResults[file], null, 2);
        // Write the results to a JSON file
        fs.writeFileSync(path.join(CONFIG_ABSOLUTE_PATH, `${file}.json`), stringified);
    });
}

compileToJson().catch(console.error);
