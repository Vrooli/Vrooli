/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DEFAULT_LANGUAGE, LlmTask } from "@local/shared";
import { expect } from "chai";
import fs from "fs";
import { RedisClientType } from "redis";
import sinon from "sinon";
import { closeRedis, initializeRedis } from "../../redisConn.js";
import { LLM_CONVERTER_LOCATION, generateTaskExec, importConverter } from "./converter.js";

describe("importConverter", () => {
    const converterFiles = fs.readdirSync(LLM_CONVERTER_LOCATION).filter(file => file.endsWith(".ts"));
    const sandbox = sinon.createSandbox();

    beforeEach(() => {
        sandbox.restore();
    });

    afterEach(() => {
        sandbox.restore();
    });

    converterFiles.forEach(async file => {
        it(`ensures the converter is correctly imported from ${file}`, async () => {
            const converter = await importConverter(file.replace(".ts", "")); // Remove file extension if needed
            expect(converter).to.exist;
            expect(typeof converter).to.equal("object");

            // Make sure each llm task has a corresponding conversion function
            Object.keys(LlmTask).forEach(action => {
                const conversionFunction = converter[action];
                expect(conversionFunction).to.exist;
                expect(typeof conversionFunction).to.equal("function");

                // Should return a valid object for some input, no matter what the input is
                const mockData = { some: "data" };
                const result = conversionFunction(mockData, "en", {});
                expect(result).to.exist;
                expect(typeof result).to.equal("object");
            });
        });
    });

    it(`falls back to ${DEFAULT_LANGUAGE} converter when a language converter is not found`, async () => {
        // Attempt to import a converter for a nonexistent language
        const converter = await importConverter("nonexistent-language");
        expect(converter).to.exist;

        // Import the default language converter for comparison
        const defaultConverter = await importConverter(DEFAULT_LANGUAGE);

        // Compare functions by converting them to strings
        const converterFuncs = Object.keys(converter).map(key => converter[key].toString());
        const defaultConverterFuncs = Object.keys(defaultConverter).map(key => defaultConverter[key].toString());

        expect(converterFuncs).to.deep.equal(defaultConverterFuncs);
    });
});

describe("generateTaskExec", () => {
    const mockUserData = { languages: ["en"] };
    let redis: RedisClientType;

    before(async () => {
        redis = await initializeRedis() as RedisClientType;
    });

    after(async () => {
        await closeRedis();
    });

    Object.keys(LlmTask).forEach((task) => {
        // NOTE: We skip "Start", since it's a special case
        if (task === "Start") return;

        it(`should create a task exec function for ${task}`, async () => {
            // @ts-ignore: Testing runtime scenario 
            const execFunc = await generateTaskExec(task, "en", mockUserData);
            expect(typeof execFunc).to.equal("function");
        });
    });

    it("should throw an error for invalid tasks", async () => {
        try {
            // @ts-ignore: Testing runtime scenario
            await generateTaskExec("InvalidTask", "en", mockUserData);
            // If we get here, no error was thrown, which is a fail
            expect.fail("Expected an error to be thrown");
        } catch (error) {
            // We expect an error to be thrown
            expect(error).to.exist;
        }
    });
});
