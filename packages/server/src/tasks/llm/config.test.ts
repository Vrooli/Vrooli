/* eslint-disable @typescript-eslint/ban-ts-comment */
import fs from "fs";
import pkg from "../../__mocks__/@prisma/client";
import { mockPrisma, resetPrismaMockData } from "../../__mocks__/prismaUtils";
import { DEFAULT_LANGUAGE, LLM_CONFIG_LOCATION, generateTaskExec, getStructuredTaskConfig, getUnstructuredTaskConfig, importCommandToTask, importConfig, importConverter, llmTasks } from "./config";

const { PrismaClient } = pkg;

jest.mock("@prisma/client");
jest.mock("../../events/logger");

describe("importConfig", () => {
    const configFiles = fs.readdirSync(LLM_CONFIG_LOCATION).filter(file => file.endsWith(".ts"));

    configFiles.forEach(async file => {
        test(`ensures all actions are present in ${file}`, async () => {
            const config = await importConfig(file);
            llmTasks.forEach(action => {
                const actionConfig = config[action];
                expect(actionConfig).toBeDefined();

                const validateStructure = (obj) => {
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
                };

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

        test(`ensures ${file} has a __construct_context function with the correct behavior`, async () => {
            const config = await importConfig(file);

            // Check for the existence of the __construct_context function
            expect(config.__construct_context).toBeDefined();
            expect(typeof config.__construct_context).toBe("function");

            // Create a mock argument with the expected structure
            const mockContextArg = {
                actions: ["action1", "action2"], // Example actions
                properties: ["property1", { name: "property2", type: "type2" }], // Example properties
                commands: { command1: "Command 1", command2: "Command 2" }, // Example commands
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

    test(`falls back to ${DEFAULT_LANGUAGE} config when a language config is not found`, async () => {
        const config = await importConfig("nonexistent-language");
        expect(config).toBeDefined();
        expect(config).toEqual(await importConfig(DEFAULT_LANGUAGE));
    });
});

describe("importConverter", () => {
    const converterFiles = fs.readdirSync(LLM_CONFIG_LOCATION).filter(file => file.endsWith(".ts"));

    converterFiles.forEach(async file => {
        test(`ensures the converter is correctly imported from ${file}`, async () => {
            const converter = await importConverter(file.replace(".ts", "")); // Remove file extension if needed
            expect(converter).toBeDefined();
            expect(typeof converter).toBe("object");

            // Make sure each llm task has a corresponding conversion function
            llmTasks.forEach(action => {
                const conversionFunction = converter[action];
                expect(conversionFunction).toBeDefined();
                expect(typeof conversionFunction).toBe("function");

                // Should return a valid object for some input, no matter what the input is
                const mockData = { some: "data" };
                const result = conversionFunction(mockData, {});
                expect(result).toBeDefined();
                expect(typeof result).toBe("object");
            });
        });
    });

    test(`falls back to ${DEFAULT_LANGUAGE} converter when a language converter is not found`, async () => {
        // Attempt to import a converter for a nonexistent language
        const converter = await importConverter("nonexistent-language");
        expect(converter).toBeDefined();

        // Import the default language converter for comparison
        const defaultConverter = await importConverter(DEFAULT_LANGUAGE);
        expect(converter.toString()).toEqual(defaultConverter.toString()); // Compare function implementations as a string
    });

    // Additional tests can be added to check for specific behaviors or properties of the converters
});

describe("importCommandToTask", () => {
    const converterFiles = fs.readdirSync(LLM_CONFIG_LOCATION).filter(file => file.endsWith(".ts"));

    converterFiles.forEach(async file => {
        test(`ensures the command-to-task conversion function is correctly imported from ${file}`, async () => {
            const converter = await importCommandToTask(file.replace(".ts", "")); // Remove file extension if needed
            expect(converter).toBeDefined();
            expect(typeof converter).toBe("function");
        });
    });
});

describe("getUnstructuredTaskConfig", () => {
    test("works for an existing language", async () => {
        const taskConfig = await getUnstructuredTaskConfig("Start", DEFAULT_LANGUAGE);
        expect(taskConfig).toBeDefined();
        // Perform additional checks on the structure of taskConfig if necessary
    });

    test("works for a language that's not in the config", async () => {
        const taskConfig = await getUnstructuredTaskConfig("RoutineAdd", "nonexistent-language");
        expect(taskConfig).toBeDefined();
        // Since it falls back to English, compare it with the English config for the same action
        const englishConfig = await getUnstructuredTaskConfig("RoutineAdd", DEFAULT_LANGUAGE);
        expect(taskConfig).toEqual(englishConfig);
    });

    test("returns an empty object for an invalid action or action that doesn't appear in the config", async () => {
        // @ts-ignore: Testing runtime scenario
        const taskConfig = await getUnstructuredTaskConfig("InvalidAction", "en");
        expect(taskConfig).toEqual({});
    });
});

describe("getStructuredTaskConfig", () => {
    test("works for an existing language", async () => {
        const taskConfig = await getStructuredTaskConfig("Start", false, DEFAULT_LANGUAGE);
        expect(taskConfig).toBeDefined();
        // Perform additional checks on the structure of taskConfig if necessary
    });

    test("works for a language that's not in the config", async () => {
        const taskConfig = await getStructuredTaskConfig("RoutineAdd", false, "nonexistent-language");
        expect(taskConfig).toBeDefined();
        // Since it falls back to English, compare it with the English config for the same action
        const englishConfig = await getStructuredTaskConfig("RoutineAdd", false, DEFAULT_LANGUAGE);
        expect(taskConfig).toEqual(englishConfig);
    });

    test("returns an empty object for an invalid action or action that doesn't appear in the config", async () => {
        // @ts-ignore: Testing runtime scenario
        const taskConfig = await getStructuredTaskConfig("InvalidAction", false, "en");
        expect(taskConfig).toEqual({});
    });

    test("works when 'force' is true", async () => {
        const taskConfig = await getStructuredTaskConfig("RoutineAdd", true, DEFAULT_LANGUAGE);
        expect(taskConfig).toBeDefined();
        // Perform additional checks on the structure of taskConfig if necessary
    });
});

describe("generateTaskExec", () => {
    let prismaMock;
    const mockUserData = { languages: ["en"] };

    beforeEach(async () => {
        jest.clearAllMocks();
        prismaMock = mockPrisma({});
        PrismaClient.injectMocks(prismaMock);
    });

    afterEach(() => {
        PrismaClient.resetMocks();
        resetPrismaMockData();
    });

    llmTasks.forEach((task) => {
        it(`should create a task exec function for ${task}`, async () => {
            // @ts-ignore: Testing runtime scenario 
            const execFunc = await generateTaskExec(task, "en", prismaMock, mockUserData);
            expect(typeof execFunc).toBe("function");
        });
    });

    it("should throw an error for invalid tasks", async () => {
        // @ts-ignore: Testing runtime scenario
        await expect(generateTaskExec("InvalidTask", "en", prismaMock, mockUserData)).rejects.toThrow();
    });
});
