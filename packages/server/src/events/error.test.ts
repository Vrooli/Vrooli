import { expect } from "chai";
import sinon from "sinon";
import { CustomError } from "./error.js";
import { logger } from "./logger.js";

describe("CustomError", () => {
    let loggerErrorStub: sinon.SinonStub;

    beforeEach(() => {
        // Create a stub for the logger.error method
        loggerErrorStub = sinon.stub(logger, "error");
    });

    afterEach(() => {
        // Restore the original logger methods after each test
        loggerErrorStub.restore();
    });

    it("should generate an error with a correct CouldNotReadObject message", () => {
        const error = new CustomError("TEST", "CouldNotReadObject");

        expect(error.message).to.match(/CouldNotReadObject: TEST-/);
        expect(loggerErrorStub.called).to.be.true;
        expect(error.code).to.equal("CouldNotReadObject");
        expect(error.trace).to.match(/TEST-/);
    });

    it("should generate an error with a correct MaxFileSizeExceeded message", () => {
        const error = new CustomError("TEST", "MaxFileSizeExceeded");

        expect(error.message).to.match(/MaxFileSizeExceeded: TEST-/);
        expect(loggerErrorStub.called).to.be.true;
        expect(error.code).to.equal("MaxFileSizeExceeded");
        expect(error.trace).to.match(/TEST-/);
    });

    it("should include additional data when provided", () => {
        const additionalData = { userId: "123", action: "upload" };
        const error = new CustomError("TEST", "CouldNotReadObject", additionalData);

        expect(error.message).to.match(/CouldNotReadObject: TEST-/);
        expect(loggerErrorStub.calledOnce).to.be.true;

        // Check that the logger was called with the additional data
        const logArgs = loggerErrorStub.firstCall.args[0];
        expect(logArgs).to.include(additionalData);
    });

    it("should provide a method to convert to ServerError", () => {
        const error = new CustomError("TEST", "CouldNotReadObject");
        const serverError = error.toServerError();

        expect(serverError).to.have.property("trace");
        expect(serverError).to.have.property("code", "CouldNotReadObject");
        expect(serverError.trace).to.equal(error.trace);
    });
});
