import { PaymentType } from "@vrooli/shared";
import type Bull from "bull";
import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import sinon from "sinon";
import * as yup from "yup";
import { logger } from "../../events/logger.js";
import { type EmailProcessPayload, emailQueue, feedbackNotifyAdmin, sendCreditCardExpiringSoon, sendMail, sendPaymentFailed, sendPaymentThankYou, sendResetPasswordLink, sendSubscriptionCanceled, sendSubscriptionEnded, sendVerificationLink } from "./queue.js";

/**
 * Validates the email task object structure.
 */
const emailSchema = yup.object().shape({
    to: yup.array().of(yup.string().email()).min(1, "The \"to\" field must contain at least one email"),
    subject: yup.string().required("Subject is required").min(1, "Subject cannot be empty"),
    text: yup.string().required("Text is required").min(1, "Text cannot be empty"),
    html: yup.string().min(1, "HTML cannot be empty").notRequired(),
});

describe("Email Queue Tests", () => {
    let loggerErrorStub;
    let loggerInfoStub;
    let queueAddStub;
    const originalSiteEmailUsername = process.env.SITE_EMAIL_USERNAME;

    beforeAll(() => {
        // Stub logger methods to prevent console output during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");

        // Set test email for admin
        process.env.SITE_EMAIL_USERNAME = "admin@example.com";
    });

    afterAll(() => {
        // Restore all stubs and original env variables
        loggerErrorStub.restore();
        loggerInfoStub.restore();
        process.env.SITE_EMAIL_USERNAME = originalSiteEmailUsername;
    });

    beforeEach(() => {
        // Create a mock job
        const mockJob = {
            id: "test-id",
            data: {} as EmailProcessPayload,
            opts: {},
            attemptsMade: 0,
        } as Bull.Job<EmailProcessPayload>;

        // Stub emailQueue's getQueue().add method
        queueAddStub = sinon.stub().resolves(mockJob);

        // Create a mock queue with the minimum required for our tests
        const mockQueue = {
            add: queueAddStub,
            // Add other required Bull.Queue properties as needed
            name: "email",
            client: {} as any,
            process: () => { },
        } as unknown as Bull.Queue<EmailProcessPayload>;

        // Replace emailQueue.getQueue with a function that returns our mock
        sinon.stub(emailQueue, "getQueue").returns(mockQueue);
    });

    afterEach(() => {
        sinon.restore();
    });

    /**
     * Helper function to assert email task properties.
     * @param expectedData The expected data object to validate against.
     */
    async function expectEmailToBeEnqueuedWith(expectedData) {
        // Verify the add method was called
        expect(queueAddStub.called).to.be.true;

        // Get the actual data passed to add
        const actualData = queueAddStub.firstCall.args[0];

        // Validate against schema
        const validatedData = await emailSchema.validate(actualData);
        expect(validatedData).to.deep.equal(actualData);

        // Check if it matches expected data
        if (expectedData) {
            if (expectedData.to) {
                expect(actualData.to).to.deep.equal(expectedData.to);
            }
            if (expectedData.subject) {
                expect(actualData.subject).to.equal(expectedData.subject);
            }
            if (expectedData.text) {
                expect(actualData.text).to.equal(expectedData.text);
            }
            // Special handling for html field, which can be undefined
            if ("html" in expectedData) {
                if (expectedData.html === undefined) {
                    expect(actualData.html).to.be.undefined;
                } else {
                    expect(actualData.html).to.equal(expectedData.html);
                }
            }
        }

        return actualData;
    }

    describe("sendMail function tests", () => {
        it("enqueues an email with all parameters provided", async () => {
            const testEmails = ["user1@example.com", "user2@example.com"];
            const subject = "Test Subject";
            const text = "Test email body";
            const html = "<p>Test email body</p>";
            const delay = 5000; // 5 seconds

            await sendMail(testEmails, subject, text, html, delay);

            await expectEmailToBeEnqueuedWith({
                to: testEmails,
                subject,
                text,
                html,
            });

            // Check if the delay option is set correctly
            const options = queueAddStub.firstCall.args[1];
            expect(options).to.deep.equal({ delay });
        });

        it("enqueues an email without the html parameter", async () => {
            const testEmails = ["user@example.com"];
            const subject = "No HTML Subject";
            const text = "Email body without HTML";

            await sendMail(testEmails, subject, text);

            // Get the actual data passed to the queue
            const actualData = queueAddStub.firstCall.args[0];

            // Check each property individually
            expect(actualData.to).to.deep.equal(testEmails);
            expect(actualData.subject).to.equal(subject);
            expect(actualData.text).to.equal(text);
            expect(actualData.html).to.be.undefined;

            // Check that default delay is 0
            const options = queueAddStub.firstCall.args[1];
            expect(options).to.deep.equal({ delay: 0 });
        });

        it("enqueues an email with a delay", async () => {
            const delay = 10000; // 10 seconds
            await sendMail(["delayed@example.com"], "Delayed Email", "This is a delayed email.", "", delay);

            // Get the actual data passed to the queue
            const actualData = queueAddStub.firstCall.args[0];

            // Check each property individually
            expect(actualData.to).to.deep.equal(["delayed@example.com"]);
            expect(actualData.subject).to.equal("Delayed Email");
            expect(actualData.text).to.equal("This is a delayed email.");
            expect(actualData.html).to.be.undefined;

            // Check if the delay option is set correctly
            const options = queueAddStub.firstCall.args[1];
            expect(options).to.deep.equal({ delay });
        });

        it("throws an error if no recipient is provided", async () => {
            expect(() => {
                sendMail([], "Subject", "Text body");
            }).to.throw();
        });
    });

    describe("Email task functions tests", () => {
        const testEmail = "valid.email@example.com";

        it("correctly enqueues email for sendResetPasswordLink function", async () => {
            const userPublicId = "userPublicId1";
            const code = "code1";
            await sendResetPasswordLink(testEmail, userPublicId, code);

            const actualData = await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });

            // Make sure that the text and html both include `${userPublicId}:${code}`
            const partialLink = `${userPublicId}:${code}`;
            expect(actualData.text).to.include(partialLink);
            expect(actualData.html).to.include(partialLink);
        });

        it("correctly enqueues email for sendVerificationLink function", async () => {
            const userPublicId = "userPublicId1";
            const code = "code1";
            await sendVerificationLink(testEmail, userPublicId, code);

            const actualData = await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });

            // Make sure that the text and html both include `${userPublicId}:${code}`
            const partialLink = `${userPublicId}:${code}`;
            expect(actualData.text).to.include(partialLink);
            expect(actualData.html).to.include(partialLink);
        });

        it("correctly enqueues feedback notification email for the admin", async () => {
            const feedbackText = "This is a feedback message.";
            const feedbackFrom = "user@example.com";
            await feedbackNotifyAdmin(feedbackText, feedbackFrom);

            await expectEmailToBeEnqueuedWith({
                to: [process.env.SITE_EMAIL_USERNAME],
                subject: "Received Vrooli Feedback!",
                text: `Feedback from ${feedbackFrom}: ${feedbackText}`,
            });
        });

        it("correctly handles anonymous feedback notification email for the admin", async () => {
            const feedbackText = "This is anonymous feedback.";
            await feedbackNotifyAdmin(feedbackText);

            await expectEmailToBeEnqueuedWith({
                to: [process.env.SITE_EMAIL_USERNAME],
                subject: "Received Vrooli Feedback!",
                text: `Feedback from anonymous: ${feedbackText}`,
            });
        });

        it("correctly enqueues email for sendPaymentThankYou function", async () => {
            await sendPaymentThankYou(testEmail, true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });
        });

        it("correctly enqueues email for sendPaymentFailed function", async () => {
            await sendPaymentFailed(testEmail, PaymentType.Donation);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });
        });

        it("correctly enqueues email for sendCreditCardExpiringSoon function", async () => {
            await sendCreditCardExpiringSoon(testEmail);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });
        });

        it("correctly enqueues email for sendSubscriptionCanceled function", async () => {
            await sendSubscriptionCanceled(testEmail);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });
        });

        it("correctly enqueues email for sendSubscriptionEnded function", async () => {
            await sendSubscriptionEnded(testEmail);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });
        });
    });
});
