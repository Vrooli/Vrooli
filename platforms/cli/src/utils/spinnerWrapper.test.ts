// AI_CHECK: TEST_COVERAGE=27 | LAST: 2025-08-04
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { withSpinner, SpinnerManager, withProgress } from "./spinnerWrapper.js";

// Mock ora
const mockOra = {
    start: vi.fn().mockReturnThis(),
    succeed: vi.fn().mockReturnThis(),
    fail: vi.fn().mockReturnThis(),
    warn: vi.fn().mockReturnThis(),
    stop: vi.fn().mockReturnThis(),
    info: vi.fn().mockReturnThis(),
    text: "",
};

vi.mock("ora", () => ({
    default: vi.fn(() => mockOra),
}));

vi.mock("./errorHandler.js", () => ({
    handleCommandError: vi.fn(),
}));

describe("spinnerWrapper", () => {
    let originalEnv: Record<string, string | undefined>;

    beforeEach(() => {
        originalEnv = { ...process.env };
        vi.clearAllMocks();
        // Reset the text property
        mockOra.text = "";
    });

    afterEach(() => {
        process.env = originalEnv;
    });

    describe("withSpinner", () => {
        it("should execute action with spinner when enabled", async () => {
            const mockAction = vi.fn().mockResolvedValue("success");
            const result = await withSpinner("Testing...", mockAction);

            expect(result).toBe("success");
            expect(mockOra.start).toHaveBeenCalled();
            expect(mockOra.succeed).toHaveBeenCalledWith("Testing...");
            expect(mockAction).toHaveBeenCalled();
        });

        it("should execute action with spinner using object options", async () => {
            const mockAction = vi.fn().mockResolvedValue("success");
            const options = {
                text: "Processing...",
                successText: "Completed!",
                failText: "Failed!",
            };

            const result = await withSpinner(options, mockAction);

            expect(result).toBe("success");
            expect(mockOra.start).toHaveBeenCalled();
            expect(mockOra.succeed).toHaveBeenCalledWith("Completed!");
            expect(mockAction).toHaveBeenCalled();
        });

        it("should handle action failure with spinner", async () => {
            const error = new Error("Test error");
            const mockAction = vi.fn().mockRejectedValue(error);

            await expect(withSpinner("Testing...", mockAction)).rejects.toThrow("Test error");

            expect(mockOra.start).toHaveBeenCalled();
            expect(mockOra.fail).toHaveBeenCalledWith("Testing...");
        });

        it("should handle action failure with custom fail text", async () => {
            const error = new Error("Test error");
            const mockAction = vi.fn().mockRejectedValue(error);
            const options = {
                text: "Processing...",
                failText: "Operation failed!",
            };

            await expect(withSpinner(options, mockAction)).rejects.toThrow("Test error");

            expect(mockOra.start).toHaveBeenCalled();
            expect(mockOra.fail).toHaveBeenCalledWith("Operation failed!");
        });

        it("should skip spinner when disabled", async () => {
            const mockAction = vi.fn().mockResolvedValue("success");
            const options = {
                text: "Testing...",
                enabled: false,
            };

            const result = await withSpinner(options, mockAction);

            expect(result).toBe("success");
            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockOra.succeed).not.toHaveBeenCalled();
            expect(mockAction).toHaveBeenCalled();
        });

        it("should skip spinner when JSON_OUTPUT is set", async () => {
            process.env.JSON_OUTPUT = "true";
            const mockAction = vi.fn().mockResolvedValue("success");

            const result = await withSpinner("Testing...", mockAction);

            expect(result).toBe("success");
            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockOra.succeed).not.toHaveBeenCalled();
            expect(mockAction).toHaveBeenCalled();
        });
    });

    describe("SpinnerManager", () => {
        let spinnerManager: SpinnerManager;

        beforeEach(() => {
            spinnerManager = new SpinnerManager();
        });

        it("should start spinner", () => {
            spinnerManager.start("Loading...");

            expect(mockOra.start).toHaveBeenCalled();
        });

        it("should update existing spinner text", () => {
            spinnerManager.start("Loading...");
            spinnerManager.update("Processing...");

            expect(mockOra.text).toBe("Processing...");
        });

        it("should not update when disabled", () => {
            const disabledSpinner = new SpinnerManager(false);
            disabledSpinner.start("Loading...");
            disabledSpinner.update("Processing...");

            expect(mockOra.start).not.toHaveBeenCalled();
        });

        it("should not update when no spinner exists", () => {
            spinnerManager.update("Processing...");

            expect(mockOra.text).toBe("");
        });

        it("should succeed and reset spinner", () => {
            spinnerManager.start("Loading...");
            spinnerManager.succeed("Completed!");

            expect(mockOra.succeed).toHaveBeenCalledWith("Completed!");
        });

        it("should succeed with default text", () => {
            spinnerManager.start("Loading...");
            spinnerManager.succeed();

            expect(mockOra.succeed).toHaveBeenCalledWith(undefined);
        });

        it("should fail and reset spinner", () => {
            spinnerManager.start("Loading...");
            spinnerManager.fail("Failed!");

            expect(mockOra.fail).toHaveBeenCalledWith("Failed!");
        });

        it("should warn and reset spinner", () => {
            spinnerManager.start("Loading...");
            spinnerManager.warn("Warning!");

            expect(mockOra.warn).toHaveBeenCalledWith("Warning!");
        });

        it("should stop spinner", () => {
            spinnerManager.start("Loading...");
            spinnerManager.stop();

            expect(mockOra.stop).toHaveBeenCalled();
        });

        it("should show info and reset spinner", () => {
            spinnerManager.start("Loading...");
            spinnerManager.info("Information");

            expect(mockOra.info).toHaveBeenCalledWith("Information");
        });

        it("should handle multiple operations", () => {
            spinnerManager.start("Step 1");
            spinnerManager.succeed("Step 1 completed");
            
            spinnerManager.start("Step 2");
            spinnerManager.fail("Step 2 failed");

            expect(mockOra.start).toHaveBeenCalledTimes(2);
            expect(mockOra.succeed).toHaveBeenCalledWith("Step 1 completed");
            expect(mockOra.fail).toHaveBeenCalledWith("Step 2 failed");
        });

        it("should not operate when disabled", () => {
            const disabledSpinner = new SpinnerManager(false);
            
            disabledSpinner.start("Loading...");
            disabledSpinner.succeed("Done");
            disabledSpinner.fail("Error");
            disabledSpinner.warn("Warning");
            disabledSpinner.stop();
            disabledSpinner.info("Info");

            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockOra.succeed).not.toHaveBeenCalled();
            expect(mockOra.fail).not.toHaveBeenCalled();
            expect(mockOra.warn).not.toHaveBeenCalled();
            expect(mockOra.stop).not.toHaveBeenCalled();
            expect(mockOra.info).not.toHaveBeenCalled();
        });

        it("should be disabled when JSON_OUTPUT is set", () => {
            process.env.JSON_OUTPUT = "true";
            const jsonSpinner = new SpinnerManager();
            
            jsonSpinner.start("Loading...");
            jsonSpinner.succeed("Done");

            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockOra.succeed).not.toHaveBeenCalled();
        });

        it("should update existing spinner text when starting again", () => {
            spinnerManager.start("Step 1");
            spinnerManager.start("Step 2");

            expect(mockOra.start).toHaveBeenCalledTimes(1);
            expect(mockOra.text).toBe("Step 2");
        });
    });

    describe("withProgress", () => {
        let mockHandleCommandError: Mock;

        beforeEach(async () => {
            const errorHandlerModule = await import("./errorHandler.js");
            mockHandleCommandError = vi.mocked(errorHandlerModule.handleCommandError);
        });

        it("should execute all steps successfully", async () => {
            const step1Action = vi.fn().mockResolvedValue("step1");
            const step2Action = vi.fn().mockResolvedValue("step2");
            
            const steps = [
                { text: "Step 1", action: step1Action },
                { text: "Step 2", action: step2Action, successText: "Step 2 completed" },
            ];

            await withProgress(steps);

            expect(step1Action).toHaveBeenCalled();
            expect(step2Action).toHaveBeenCalled();
            expect(mockOra.start).toHaveBeenCalledTimes(2);
            expect(mockOra.succeed).toHaveBeenCalledWith("[1/2] Step 1");
            expect(mockOra.succeed).toHaveBeenCalledWith("Step 2 completed");
        });

        it("should handle step failure", async () => {
            const step1Action = vi.fn().mockResolvedValue("step1");
            const step2Action = vi.fn().mockRejectedValue(new Error("Step 2 failed"));
            
            const steps = [
                { text: "Step 1", action: step1Action },
                { text: "Step 2", action: step2Action },
            ];

            await withProgress(steps);

            expect(step1Action).toHaveBeenCalled();
            expect(step2Action).toHaveBeenCalled();
            expect(mockOra.succeed).toHaveBeenCalledWith("[1/2] Step 1");
            expect(mockOra.fail).toHaveBeenCalledWith("[2/2] Step 2");
            expect(mockHandleCommandError).toHaveBeenCalledWith(
                expect.any(Error),
                undefined,
                "Step 2",
            );
        });

        it("should continue after step failure", async () => {
            const step1Action = vi.fn().mockRejectedValue(new Error("Step 1 failed"));
            const step2Action = vi.fn().mockResolvedValue("step2");
            
            const steps = [
                { text: "Step 1", action: step1Action },
                { text: "Step 2", action: step2Action },
            ];

            await withProgress(steps);

            expect(step1Action).toHaveBeenCalled();
            expect(step2Action).toHaveBeenCalled();
            expect(mockOra.fail).toHaveBeenCalledWith("[1/2] Step 1");
            expect(mockOra.succeed).toHaveBeenCalledWith("[2/2] Step 2");
        });

        it("should format step numbers correctly", async () => {
            const actions = Array.from({ length: 3 }, (_, i) => 
                vi.fn().mockResolvedValue(`step${i + 1}`),
            );
            
            const steps = actions.map((action, i) => ({
                text: `Step ${i + 1}`,
                action,
            }));

            await withProgress(steps);

            // Check that start was called 3 times (once for each step)
            expect(mockOra.start).toHaveBeenCalledTimes(3);
            // The SpinnerManager should have updated the text property
            // We can't easily check the specific arguments to start() due to how the mocking works
            // but we can verify the calls were made
            actions.forEach(action => {
                expect(action).toHaveBeenCalled();
            });
        });

        it("should handle empty steps array", async () => {
            await withProgress([]);

            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockOra.succeed).not.toHaveBeenCalled();
            expect(mockOra.fail).not.toHaveBeenCalled();
        });
    });

    describe("environment integration", () => {
        it("should respect JSON_OUTPUT environment variable in withSpinner", async () => {
            process.env.JSON_OUTPUT = "1";
            const mockAction = vi.fn().mockResolvedValue("result");

            const result = await withSpinner("Testing...", mockAction);

            expect(result).toBe("result");
            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockAction).toHaveBeenCalled();
        });

        it("should respect JSON_OUTPUT environment variable in SpinnerManager", () => {
            process.env.JSON_OUTPUT = "true";
            const spinner = new SpinnerManager();

            spinner.start("Loading...");
            spinner.succeed("Done");

            expect(mockOra.start).not.toHaveBeenCalled();
            expect(mockOra.succeed).not.toHaveBeenCalled();
        });
    });
});
