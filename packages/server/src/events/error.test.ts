import { expect, describe, it, beforeEach, afterEach, vi } from "vitest";

import { CustomError } from "./error.js";
import { logger } from "./logger.js";

describe("CustomError", () => {
    let loggerErrorStub: any;

    beforeEach(() => {
        // Create a mock for the logger.error method
        loggerErrorStub = vi.spyOn(logger, "error").mockImplementation(() => {});
    });

    afterEach(() => {
        // Restore the original logger methods after each test
        vi.restoreAllMocks();
    });

    it("should generate an error with a correct CouldNotReadObject message", () => {
        const error = new CustomError("TEST", "CouldNotReadObject");

        expect(error.message).toMatch(/CouldNotReadObject: TEST-/);
        expect(loggerErrorStub).toHaveBeenCalled();
        expect(error.code).toBe("CouldNotReadObject");
        expect(error.trace).toMatch(/TEST-/);
    });

    it("should generate an error with a correct MaxFileSizeExceeded message", () => {
        const error = new CustomError("TEST", "MaxFileSizeExceeded");

        expect(error.message).toMatch(/MaxFileSizeExceeded: TEST-/);
        expect(loggerErrorStub).toHaveBeenCalled();
        expect(error.code).toBe("MaxFileSizeExceeded");
        expect(error.trace).toMatch(/TEST-/);
    });

    it("should include additional data when provided", () => {
        const additionalData = { userId: "123", action: "upload" };
        const error = new CustomError("TEST", "CouldNotReadObject", additionalData);

        expect(error.message).toMatch(/CouldNotReadObject: TEST-/);
        expect(loggerErrorStub).toHaveBeenCalledOnce();

        // Check that the logger was called with the additional data spread into the log object
        const logArgs = loggerErrorStub.mock.calls[0][0];
        expect(logArgs).toMatchObject({
            userId: "123",
            action: "upload",
            msg: "CouldNotReadObject",
            trace: expect.stringMatching(/TEST-/)
        });
    });

    it("should provide a method to convert to ServerError", () => {
        const error = new CustomError("TEST", "CouldNotReadObject");
        const serverError = error.toServerError();

        expect(serverError).toHaveProperty("trace");
        expect(serverError).toHaveProperty("code", "CouldNotReadObject");
        expect(serverError.trace).toBe(error.trace);
    });
});
