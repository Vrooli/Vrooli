import { expect } from "chai";
import { stub } from "sinon";
import { noopSubmit } from "./noop.js";

describe("noop utilities", () => {
    describe("noopSubmit", () => {
        let consoleWarnStub: any;
        let setSubmittingStub: any;

        beforeEach(() => {
            // Stub console.warn to prevent output during tests
            consoleWarnStub = stub(console, "warn");
            setSubmittingStub = stub();
        });

        afterEach(() => {
            // Restore the original console.warn
            consoleWarnStub.restore();
        });

        it("should call console.warn with the provided values", () => {
            const values = { username: "testuser", email: "test@example.com" };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledOnce).to.be.true;
            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", values)).to.be.true;
        });

        it("should call setSubmitting with false", () => {
            const values = { field: "value" };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(setSubmittingStub.calledOnce).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should handle null values", () => {
            const values = null;
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", null)).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should handle undefined values", () => {
            const values = undefined;
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", undefined)).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should handle empty object values", () => {
            const values = {};
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", {})).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should handle complex nested values", () => {
            const values = {
                user: {
                    name: "John Doe",
                    preferences: {
                        theme: "dark",
                        notifications: true,
                    },
                },
                items: [1, 2, 3],
            };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", values)).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should handle array values", () => {
            const values = [1, 2, 3, 4, 5];
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", values)).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should handle string values", () => {
            const values = "unexpected string submission";
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub.calledWith("Formik onSubmit called unexpectedly with values:", values)).to.be.true;
            expect(setSubmittingStub.calledWith(false)).to.be.true;
        });

        it("should work when setSubmitting throws an error", () => {
            const values = { test: "value" };
            const errorMessage = "setSubmitting failed";
            const throwingSetSubmitting = stub().throws(new Error(errorMessage));
            const helpers = { setSubmitting: throwingSetSubmitting };

            expect(() => noopSubmit(values, helpers)).to.throw(errorMessage);
            expect(consoleWarnStub.calledOnce).to.be.true;
        });

        it("should be called multiple times without issues", () => {
            const values1 = { call: 1 };
            const values2 = { call: 2 };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values1, helpers);
            noopSubmit(values2, helpers);

            expect(consoleWarnStub.calledTwice).to.be.true;
            expect(setSubmittingStub.calledTwice).to.be.true;
            expect(consoleWarnStub.firstCall.args[1]).to.deep.equal(values1);
            expect(consoleWarnStub.secondCall.args[1]).to.deep.equal(values2);
        });
    });
});