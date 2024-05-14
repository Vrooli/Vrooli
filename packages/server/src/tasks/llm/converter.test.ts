/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DEFAULT_LANGUAGE, LlmTask } from "@local/shared";
import fs from "fs";
import pkg from "../../__mocks__/@prisma/client";
import { LLM_CONVERTER_LOCATION, generateTaskExec, importConverter } from "./converter";

const { PrismaClient } = pkg;

jest.mock("../../events/logger");

describe("importConverter", () => {
    const converterFiles = fs.readdirSync(LLM_CONVERTER_LOCATION).filter(file => file.endsWith(".ts"));

    converterFiles.forEach(async file => {
        test(`ensures the converter is correctly imported from ${file}`, async () => {
            const converter = await importConverter(file.replace(".ts", "")); // Remove file extension if needed
            expect(converter).toBeDefined();
            expect(typeof converter).toBe("object");

            // Make sure each llm task has a corresponding conversion function
            Object.keys(LlmTask).forEach(action => {
                const conversionFunction = converter[action];
                expect(conversionFunction).toBeDefined();
                expect(typeof conversionFunction).toBe("function");

                // Should return a valid object for some input, no matter what the input is
                const mockData = { some: "data" };
                const result = conversionFunction(mockData, "en", {});
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

describe("generateTaskExec", () => {
    const mockUserData = { languages: ["en"] };

    beforeEach(async () => {
        jest.clearAllMocks();
        PrismaClient.injectData({});
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    Object.keys(LlmTask).forEach((task) => {
        // NOTE: We skip "Start", since it's a special case
        if (task === "Start") return;

        it(`should create a task exec function for ${task}`, async () => {
            // @ts-ignore: Testing runtime scenario 
            const execFunc = await generateTaskExec(task, "en", mockUserData);
            expect(typeof execFunc).toBe("function");
        });
    });

    it("should throw an error for invalid tasks", async () => {
        // @ts-ignore: Testing runtime scenario
        await expect(generateTaskExec("InvalidTask", "en", mockUserData)).rejects.toThrow();
    });
});
