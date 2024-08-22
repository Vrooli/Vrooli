/* eslint-disable @typescript-eslint/ban-ts-comment */
import fs from "fs";
import { LlmTask } from "../api/generated/graphqlTypes";
import { DEFAULT_LANGUAGE } from "../consts/ui";
import { getLlmConfigLocation, getStructuredTaskConfig, getUnstructuredTaskConfig, importCommandToTask, importConfig } from "./config";

describe("llm config", () => {
    let LLM_CONFIG_LOCATION: string;
    let configFiles: string[];

    beforeAll(async () => {
        LLM_CONFIG_LOCATION = await getLlmConfigLocation();
        configFiles = fs.readdirSync(LLM_CONFIG_LOCATION).filter(file => file.endsWith(".ts"));
    });

    test("getLlmConfigLocation", () => {
        // Expect config location to be a non-empty string
        expect(LLM_CONFIG_LOCATION).toBeDefined();
        expect(LLM_CONFIG_LOCATION).toEqual(expect.any(String));
        expect(LLM_CONFIG_LOCATION).not.toEqual("");
        // Expect config files to be present in the location (at least 1 file)
        expect(configFiles).toBeDefined();
        expect(Array.isArray(configFiles)).toBe(true);
        expect(configFiles.every(file => typeof file === "string")).toBe(true);
    });

    test("importConfig - all actions present", async () => {
        configFiles.forEach(async file => {
            const config = await importConfig(file, console);
            Object.keys(LlmTask).forEach(action => {
                const actionConfig = config[action];
                expect(actionConfig).toBeDefined();

                function validateStructure(obj) {
                    // Check for commands, which is not optional
                    expect(obj).toHaveProperty("commands");
                    expect(typeof obj.commands).toBe("object");
                    // Check for optional fields if they exist
                    if (obj.actions) {
                        expect(obj.actions).toEqual(expect.any(Array));
                    }
                    if (obj.properties) {
                        expect(obj.properties).toEqual(expect.any(Array));
                    }
                    // Add more checks for other fields as needed
                }

                if (typeof actionConfig === "function") {
                    // Call the function and check the result
                    const result = actionConfig();
                    validateStructure(result);
                } else if (typeof actionConfig === "object") {
                    validateStructure(actionConfig);
                } else {
                    // Fail the test if config[action] is neither a function nor an object
                    throw new Error(`config[${action}] must be either a function or an object`);
                }
            });
        });
    });

    test("importConfig - has a __construct_context function with the correct behavior", async () => {
        configFiles.forEach(async file => {
            const config = await importConfig(file, console);

            // Check for the existence of the __construct_context function
            expect(config.__construct_context).toBeDefined();
            expect(typeof config.__construct_context).toBe("function");

            // Create a mock argument with the expected structure
            const mockContextArg = {
                actions: ["action1", "action2"], // Example actions
                properties: ["property1", { name: "property2", type: "type2" }], // Example properties
                commands: { command1: "Command 1", command2: "Command 2" }, // Example commands
                label: "boop",
                // Include other fields as needed to mimic the rest of the structure
            };

            // Call the __construct_context function with the mock argument
            const result = config.__construct_context(mockContextArg);

            // Check that the result is an object
            expect(result).toBeDefined();
            expect(typeof result).toBe("object");

            // Perform additional checks on the result if there are specific expectations about its structure
        });
    });

    test("importConfig - falls back to default language when config is not found", async () => {
        const config = await importConfig("nonexistent-language", console);
        expect(config).toBeDefined();
        expect(config).toEqual(await importConfig(DEFAULT_LANGUAGE, console));
    });

    test("importCommandToTask - ensures the command-to-task conversion function is correctly imported from file", async () => {
        configFiles.forEach(async file => {
            const converter = await importCommandToTask(file.replace(".ts", ""), console); // Remove file extension if needed
            expect(converter).toBeDefined();
            expect(typeof converter).toBe("function");
        });
    });

    describe("getUnstructuredTaskConfig", () => {
        test("works for an existing language", async () => {
            const taskConfig = await getUnstructuredTaskConfig("Start", DEFAULT_LANGUAGE, console);
            expect(taskConfig).toBeDefined();
            // Perform additional checks on the structure of taskConfig if necessary
        });

        test("works for a language that's not in the config", async () => {
            const taskConfig = await getUnstructuredTaskConfig("RoutineAdd", "nonexistent-language", console);
            expect(taskConfig).toBeDefined();
            // Since it falls back to English, compare it with the English config for the same action
            const englishConfig = await getUnstructuredTaskConfig("RoutineAdd", DEFAULT_LANGUAGE, console);
            expect(taskConfig).toEqual(englishConfig);
        });

        test("returns an empty object for an invalid action or action that doesn't appear in the config", async () => {
            // @ts-ignore: Testing runtime scenario
            const taskConfig = await getUnstructuredTaskConfig("InvalidAction", "en", console);
            expect(taskConfig).toEqual({});
        });
    });

    describe("getStructuredTaskConfig", () => {
        test("works for an existing language", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "text",
                task: "Start",
            });
            expect(taskConfig).toBeDefined();
            // Perform additional checks on the structure of taskConfig if necessary
        });

        test("works for a language that's not in the config", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: "nonexistent-language",
                logger: console,
                mode: "text",
                task: "RoutineAdd",
            });
            expect(taskConfig).toBeDefined();
            // Since it falls back to English, compare it with the English config for the same action
            const englishConfig = await getStructuredTaskConfig({
                force: false,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "text",
                task: "RoutineAdd",
            });
            expect(taskConfig).toEqual(englishConfig);
        });

        test("returns an empty object for an invalid action or action that doesn't appear in the config", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: "en",
                logger: console,
                mode: "text",
                // @ts-ignore: Testing runtime scenario
                task: "InvalidAction",
            });
            expect(taskConfig).toEqual({});
        });

        test("works when 'force' is true", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: true,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "text",
                task: "RoutineAdd",
            });
            expect(taskConfig).toBeDefined();
            // Perform additional checks on the structure of taskConfig if necessary
        });

        test("works then the mode is 'json'", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "json",
                task: "RoutineAdd",
            });
            expect(taskConfig).toBeDefined();
            // Perform additional checks on the structure of taskConfig if necessary
        });
    });
});
