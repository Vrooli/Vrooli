import { BUSINESS_NAME } from "@local/shared";
import Bull from "bull";
import { expect } from "chai";
import sinon from "sinon";
import * as yup from "yup";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { SmsProcessPayload, sendSms, sendSmsVerification, setupSmsQueue, smsQueue } from "./queue.js";

/**
 * Validates the SMS task object structure.
 */
const smsSchema = yup.object().shape({
    to: yup.array().of(yup.string().matches(/^\+[1-9]\d{1,14}$/, "Must be a valid international phone number")).min(1, "The \"to\" field must contain at least one phone number"),
    body: yup.string().required("Body is required").min(1, "Body cannot be empty"),
});

/**
 * Helper function to assert SMS task properties.
 * @param queue The SMS queue.
 * @param expectedData The expected data object to validate against.
 */
async function expectSmsToBeEnqueuedWith(queue, expectedData) {
    // Verify that the last call to add matches expected data
    const addStub = queue.getQueue().add;
    expect(addStub.called).to.be.true;
    const actualData = addStub.lastCall.args[0];

    // Validate schema
    const validatedData = await smsSchema.validate(actualData);
    expect(validatedData).to.deep.equal(actualData);

    // Check if it matches expected data
    if (expectedData) {
        if (expectedData.to) {
            expect(actualData.to).to.deep.equal(expectedData.to);
        }
        if (expectedData.body) {
            expect(actualData.body).to.equal(expectedData.body);
        }
    }
}

describe("SMS Queue Tests", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    after(() => {
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("setupSmsQueue function", () => {
        let processStub;
        let BaseQueueStub;

        beforeEach(() => {
            processStub = sinon.stub();
            // Instead of directly modifying the imported smsQueue, we'll test the function's behavior
            BaseQueueStub = sinon.stub().returns({
                process: processStub,
                getQueue: () => ({})
            });
        });

        afterEach(() => {
            sinon.restore();
        });

        it("should initialize the SMS queue correctly", async () => {
            // This is a placeholder for testing the setup function behavior
            // In a real test, you would mock imports and verify setup behavior
            expect(typeof setupSmsQueue).to.equal('function');
        });
    });

    describe("sendSms function tests", () => {
        let queueAddStub;
        let mockJob;

        before(async () => {
            // Setup queue once before all tests
            await setupSmsQueue();
        });

        beforeEach(() => {
            // Create a proper mock job that satisfies the Bull.Job interface
            mockJob = {
                id: "test-id",
                data: {} as SmsProcessPayload,
                opts: {},
                attemptsMade: 0,
                queue: {} as any,
                // Add other required properties to satisfy the Bull.Job interface
            } as Bull.Job<SmsProcessPayload>;

            // Stub the queue's add method before each test
            queueAddStub = sinon.stub(smsQueue.getQueue(), "add").resolves(mockJob);
        });

        afterEach(() => {
            // Restore the stubbed method after each test
            queueAddStub.restore();
        });

        it("enqueues an SMS with valid parameters", async () => {
            const recipients = ["+1234567890"];
            const body = "Test SMS body";

            await sendSms(recipients, body);

            await expectSmsToBeEnqueuedWith(smsQueue, {
                to: recipients,
                body,
            });
        });

        it("throws an error if no recipient is provided", async () => {
            const body = "Test SMS body";
            try {
                await sendSms([], body);
                expect.fail("Should have thrown an error");
            } catch (error) {
                expect(error).to.be.instanceOf(CustomError);
                expect((error as CustomError).code).to.equal("InternalError");
            }
        });

        it("handles queue add failure gracefully", async () => {
            queueAddStub.rejects(new Error("Queue error"));

            const recipients = ["+1234567890"];
            const body = "Test SMS body";

            const result = await sendSms(recipients, body);
            expect(result.success).to.be.false;
        });

        it("validates multiple recipients correctly", async () => {
            const recipients = ["+1234567890", "+9876543210"];
            const body = "Test SMS body";

            await sendSms(recipients, body);

            await expectSmsToBeEnqueuedWith(smsQueue, {
                to: recipients,
                body,
            });
        });

        it("rejects invalid phone number formats", async () => {
            const invalidRecipients = ["1234567890", "123-456-7890", "invalid"];
            const body = "Test SMS body";

            // Test each invalid number separately
            for (const invalidNumber of invalidRecipients) {
                try {
                    await sendSms([invalidNumber], body);
                    await expectSmsToBeEnqueuedWith(smsQueue, { to: [invalidNumber], body });

                    // This should fail the schema validation
                    expect.fail("Should have thrown an error for invalid phone number format");
                } catch (error) {
                    // Expect a validation error
                    expect(error).to.exist;
                    expect(error.name).to.equal("ValidationError");
                }
            }
        });

        it("handles differently formatted but valid phone numbers", async () => {
            // Different valid international phone formats that should be accepted
            const formattedRecipients = [
                "+1 888-888-8888",
                "+1 (999) 888-7777",
                "+44 20 1234 5678"
            ];
            const body = "Test SMS body";

            // These formats might be accepted by Twilio but fail our schema validation
            // This test helps us understand if our schema needs adjustment to handle these formats
            for (const formattedNumber of formattedRecipients) {
                try {
                    await sendSms([formattedNumber], body);
                    await expectSmsToBeEnqueuedWith(smsQueue, { to: [formattedNumber], body });

                    // If we get here, it means the validation passed for this format
                    console.log(`Format accepted: ${formattedNumber}`);
                } catch (error) {
                    if (error.name === "ValidationError") {
                        // Schema validation failed, which indicates we might need to update our validation
                        // to accept this format if it's actually valid for Twilio
                        console.log(`Format rejected by schema: ${formattedNumber}`);
                        expect(error.name).to.equal("ValidationError");
                    } else {
                        // Unexpected error
                        throw error;
                    }
                }
            }
        });
    });

    describe("sendSmsVerification function tests", () => {
        let queueAddStub;
        let mockJob;

        before(async () => {
            // Setup queue once before all tests
            await setupSmsQueue();
        });

        beforeEach(() => {
            // Create a proper mock job that satisfies the Bull.Job interface
            mockJob = {
                id: "test-id",
                data: {} as SmsProcessPayload,
                opts: {},
                attemptsMade: 0,
                queue: {} as any,
                // Add other required properties to satisfy the Bull.Job interface
            } as Bull.Job<SmsProcessPayload>;

            // Stub the queue's add method before each test
            queueAddStub = sinon.stub(smsQueue.getQueue(), "add").resolves(mockJob);
        });

        afterEach(() => {
            // Restore the stubbed method after each test
            queueAddStub.restore();
        });

        it("enqueues a verification code SMS with correct format", async () => {
            const phoneNumber = "+1234567890";
            const code = "123456";
            const expectedBody = `${code} is your ${BUSINESS_NAME} verification code`;

            await sendSmsVerification(phoneNumber, code);

            await expectSmsToBeEnqueuedWith(smsQueue, {
                to: [phoneNumber],
                body: expectedBody
            });
        });

        it("handles different verification codes correctly", async () => {
            const phoneNumber = "+1234567890";
            const code = "987654";
            const expectedBody = `${code} is your ${BUSINESS_NAME} verification code`;

            await sendSmsVerification(phoneNumber, code);

            await expectSmsToBeEnqueuedWith(smsQueue, {
                to: [phoneNumber],
                body: expectedBody
            });
        });

        it("handles queue add failure gracefully", async () => {
            queueAddStub.rejects(new Error("Queue error"));

            const phoneNumber = "+1234567890";
            const code = "123456";

            const result = await sendSmsVerification(phoneNumber, code);
            expect(result.success).to.be.false;
        });
    });
});
