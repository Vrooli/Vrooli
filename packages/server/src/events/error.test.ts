import { expect } from "chai";
import { logger as mockLogger } from "./__mocks__/logger.js";
import { CustomError } from "./error.js";
import { logger } from "./logger.js";

jest.mock("i18next");

describe("CustomError", () => {
    let originalLoggerMethods;

    beforeEach(() => {
        // Save the original logger methods
        originalLoggerMethods = { ...logger };

        // Replace the logger methods with mocks
        Object.assign(logger, mockLogger);

        // Clear all mock calls
        jest.clearAllMocks();
    });

    afterEach(() => {
        // Restore the original logger methods after each test
        Object.assign(logger, originalLoggerMethods);
    });

    it("should generate an error with a correct message", () => {
        const error = new CustomError("TEST", "CouldNotReadObject");

        expect(error.message).to.match(/CouldNotReadObject: TEST-/);
        expect(logger.error).toHaveBeenCalled();
    });

    it("should generate an error with a correct message", () => {
        const error = new CustomError("TEST", "MaxFileSizeExceeded");

        expect(error.message).to.match(/MaxFileSizeExceeded: TEST-/);
        expect(logger.error).toHaveBeenCalled();
    });
});
