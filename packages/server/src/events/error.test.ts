import { CustomError } from "./error";
import { logger } from "./logger";

jest.mock("i18next");
jest.mock("./logger");

describe("CustomError", () => {
    beforeEach(() => {
        // Clear all instances and calls to constructor and all methods:
        jest.clearAllMocks();
    });

    it("should generate an error with a correct message", () => {
        const error = new CustomError("TEST", "CouldNotReadObject", ["en"]);

        expect(error.message).toMatch(/CouldNotReadObject: TEST-/);
        expect(logger.error).toHaveBeenCalled();
    });

    it("should generate an error with a correct message", () => {
        const error = new CustomError("TEST", "MaxFileSizeExceeded", ["en"]);

        expect(error.message).toMatch(/MaxFileSizeExceeded: TEST-/);
        expect(logger.error).toHaveBeenCalled();
    });
});
