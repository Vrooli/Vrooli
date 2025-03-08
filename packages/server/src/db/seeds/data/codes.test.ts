import { CodeVersionConfig } from "@local/shared";
import { expect } from "chai";
import { SandboxChildProcessManager } from "../../../tasks/sandbox/sandboxWorkerManager.js";
import { data } from "./codes.js";

type WorkerManager = SandboxChildProcessManager;

// Custom timeouts to speed up tests
const IDLE_TIMEOUT_MS = 100;
const JOB_TIMEOUT_MS = 300;

/**
 * Test suite for validating seeded code test cases.
 */
describe("Seeded Code Tests", () => {
    // Iterate over each code object in the seed data
    data.forEach((codeData, index) => {
        describe(`Code ${index + 1}: ${codeData.shape.id}`, () => {
            // Iterate over each version of the code object
            codeData.shape.versions.forEach((version, vIndex) => {
                describe(`Version ${vIndex + 1}: ${version.shape.versionLabel}`, () => {
                    let manager: WorkerManager;
                    let testResults: Awaited<ReturnType<CodeVersionConfig["runTestCases"]>>;

                    // Run all test cases before the individual assertions
                    before(async () => {
                        manager = new SandboxChildProcessManager({ idleTimeoutMs: IDLE_TIMEOUT_MS, jobTimeoutMs: JOB_TIMEOUT_MS });
                        const config = new CodeVersionConfig({
                            data: version.shape.data,
                            content: String.raw`${version.shape.content}`,
                            codeLanguage: version.shape.codeLanguage,
                        });
                        testResults = await config.runTestCases(manager.runUserCode.bind(manager));
                    });

                    after(async () => {
                        await manager.terminate();
                    });

                    // Create an individual test for each test case
                    version.shape.data.testCases.forEach((testCase, tIndex) => {
                        const testDesc = testCase.description || `Test case ${tIndex + 1}`;
                        it(testDesc, () => {
                            const result = testResults[tIndex];
                            if (!result.passed) {
                                console.error(`Error: ${result.error}`);
                                console.error(`Actual output: ${JSON.stringify(result.actualOutput)}`);
                                console.error(`Expected output: ${JSON.stringify(testCase.expectedOutput)}`);
                            }
                            expect(result.passed, `Test failed: ${result.error || "Output mismatch"}`).to.be.true;
                        });
                    });
                });
            });
        });
    });
});
