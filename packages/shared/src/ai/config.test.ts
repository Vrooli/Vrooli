/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import fs from "fs";
import sinon from "sinon";
import { LlmTask } from "../api/types.js";
import { DEFAULT_LANGUAGE } from "../consts/ui.js";
import { getAIConfigLocation, getStructuredTaskConfig, getUnstructuredTaskConfig, importAITaskBuilder, importAITaskConfig, importCommandToTask } from "./config.js";

describe("llm config", () => {
    let LLM_CONFIG_LOCATION: string;
    let configFiles: string[];
    let consoleErrorStub: sinon.SinonStub;

    before(async () => {
        LLM_CONFIG_LOCATION = (await getAIConfigLocation()).location;
        configFiles = fs.readdirSync(LLM_CONFIG_LOCATION).filter(file => file.endsWith(".ts"));
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("getLlmConfigLocation", () => {
        // Expect config location to be a non-empty string
        expect(LLM_CONFIG_LOCATION).to.exist;
        expect(LLM_CONFIG_LOCATION).to.be.a("string").and.not.empty;
        // Expect config files to be present in the location (at least 1 file)
        expect(configFiles).to.exist;
        expect(Array.isArray(configFiles)).to.be.true;
        expect(configFiles.every(file => typeof file === "string")).to.be.true;
    });

    describe("importConfig", () => {
        it("all tasks present", async () => {
            configFiles.forEach(async file => {
                const config = await importAITaskConfig(file, console);
                Object.keys(LlmTask).forEach(action => {
                    const actionConfig = config[action];
                    expect(actionConfig).to.exist;

                    function validateStructure(obj: Record<string, unknown>) {
                        // Check for `commands`, which is not optional
                        expect(obj).to.have.property("commands");
                        expect(obj.commands).to.be.an("object");
                        // Check for optional fields if they exist
                        if (obj.actions) {
                            expect(obj.actions).to.be.an("array");
                        }
                        if (obj.properties) {
                            expect(obj.properties).to.be.an("array");
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
        it("falls back to default language when not found", async () => {
            const config = await importAITaskConfig("nonexistent-language", console);
            expect(config).to.exist;
            const defaultConfig = await importAITaskConfig(DEFAULT_LANGUAGE, console);
            expect(config).to.deep.equal(defaultConfig);
        });
    });

    describe("importBuilder", () => {
        it("has __construct_* functions with correct behavior", async () => {
            configFiles.forEach(async file => {
                const builder = await importAITaskBuilder(file, console);

                // Check for the existence of the required functions
                const expectedFunctions = [
                    "__construct_context_text",
                    "__construct_context_text_force",
                    "__construct_context_json",
                ];
                expectedFunctions.forEach(func => {
                    expect(builder).to.have.property(func).that.is.a("function");
                });

                // Create a mock argument with the expected structure
                const mockContextArg = {
                    actions: ["action1", "action2"], // Example actions
                    properties: ["property1", { name: "property2", type: "type2" }], // Example properties
                    commands: { command1: "Command 1", command2: "Command 2" }, // Example commands
                    label: "boop",
                    // Include other fields as needed to mimic the rest of the structure
                };

                // Call each of the __construct_* functions with the mock argument
                expectedFunctions.forEach(func => {
                    const result = builder[func](mockContextArg);

                    // Check that the result is an object
                    expect(result).to.exist;
                    expect(result).to.be.an("object");

                    // Perform additional checks on the result if there are specific expectations about its structure
                });
            });
        });
        it("falls back to default language when not found", async () => {
            const builder = await importAITaskBuilder("nonexistent-language", console);
            expect(builder).to.exist;
            const defaultBuilder = await importAITaskBuilder(DEFAULT_LANGUAGE, console);
            expect(builder).to.deep.equal(defaultBuilder);
        });
    });

    it("importCommandToTask returns a valid function without logging any errors", async () => {
        for (const file of configFiles) {
            const converter = await importCommandToTask(file.replace(".ts", ""), console);
            expect(converter).to.exist;
            expect(converter).to.be.a("function");
            expect(consoleErrorStub.called).to.be.false;
        }
    });

    describe("getUnstructuredTaskConfig", () => {
        it("works for an existing language", async () => {
            const taskConfig = await getUnstructuredTaskConfig("Start", DEFAULT_LANGUAGE, console);
            expect(taskConfig).to.exist;
            // Perform additional checks on the structure of taskConfig if necessary
        });

        it("works for a language that's not in the config", async () => {
            const taskConfig = await getUnstructuredTaskConfig("RoutineAdd", "nonexistent-language", console);
            expect(taskConfig).to.exist;
            // Since it falls back to English, compare it with the English config for the same action
            const englishConfig = await getUnstructuredTaskConfig("RoutineAdd", DEFAULT_LANGUAGE, console);
            expect(taskConfig).to.deep.equal(englishConfig);
        });

        it("returns an empty object for an invalid action or action that doesn't appear in the config", async () => {
            // @ts-ignore: Testing runtime scenario
            const taskConfig = await getUnstructuredTaskConfig("InvalidAction", "en", console);
            expect(taskConfig).to.deep.equal({});
        });
    });

    describe("getStructuredTaskConfig", () => {
        it("works for an existing language", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "text",
                task: "Start",
            });
            expect(taskConfig).to.exist;
            // Perform additional checks on the structure of taskConfig if necessary
        });

        it("works for a language that's not in the config", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: "nonexistent-language",
                logger: console,
                mode: "text",
                task: "RoutineAdd",
            });
            expect(taskConfig).to.exist;
            // Since it falls back to English, compare it with the English config for the same action
            const englishConfig = await getStructuredTaskConfig({
                force: false,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "text",
                task: "RoutineAdd",
            });
            expect(taskConfig).to.deep.equal(englishConfig);
        });

        it("returns an empty object for an invalid action or action that doesn't appear in the config", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: "en",
                logger: console,
                mode: "text",
                // @ts-ignore: Testing runtime scenario
                task: "InvalidAction",
            });
            expect(taskConfig).to.deep.equal({});
        });

        it("works when 'force' is true", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: true,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "text",
                task: "RoutineAdd",
            });
            expect(taskConfig).to.exist;
            // Perform additional checks on the structure of taskConfig if necessary
        });

        it("works then the mode is 'json'", async () => {
            const taskConfig = await getStructuredTaskConfig({
                force: false,
                language: DEFAULT_LANGUAGE,
                logger: console,
                mode: "json",
                task: "RoutineAdd",
            });
            expect(taskConfig).to.exist;
            // Perform additional checks on the structure of taskConfig if necessary
        });
    });
});
