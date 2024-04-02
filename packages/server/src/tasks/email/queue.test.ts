import { PaymentType } from "@local/shared";
import * as yup from "yup";
import Bull from "../../__mocks__/bull";
import { feedbackNotifyAdmin, sendCreditCardExpiringSoon, sendMail, sendPaymentFailed, sendPaymentThankYou, sendResetPasswordLink, sendSubscriptionCanceled, sendSubscriptionEnded, sendVerificationLink, setupEmailQueue } from "./queue";

/**
 * Validates the email task object structure.
 */
const emailSchema = yup.object().shape({
    to: yup.array().of(yup.string().email()).min(1, "The \"to\" field must contain at least one email"),
    subject: yup.string().required("Subject is required").min(1, "Subject cannot be empty"),
    text: yup.string().required("Text is required").min(1, "Text cannot be empty"),
    html: yup.string().min(1, "HTML cannot be empty").notRequired(),
});

/**
 * Helper function to assert email task properties.
 * @param mockQueue The mocked email queue.
 * @param expectedData The expected data object to validate against.
 */
const expectEmailToBeEnqueuedWith = async (mockQueue, expectedData) => {
    // Check if the add function was called
    expect(mockQueue.add).toHaveBeenCalled();

    // Extract the actual data passed to the mock
    const actualData = mockQueue.add.mock.calls[0][0];

    // Validate the actual data against the schema
    await expect(emailSchema.validate(actualData)).resolves.toEqual(actualData);

    // If there's expected data, match it against the actual data
    if (expectedData) {
        expect(actualData).toMatchObject(expectedData);
    }
};

describe("sendMail function tests", () => {
    let emailQueue;

    beforeAll(async () => {
        emailQueue = new Bull("email");
        await setupEmailQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
    });

    it("enqueues an email with all parameters provided", async () => {
        const testEmails = ["user1@example.com", "user2@example.com"];
        const subject = "Test Subject";
        const text = "Test email body";
        const html = "<p>Test email body</p>";
        const delay = 5000; // 5 seconds

        sendMail(testEmails, subject, text, html, delay);

        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: testEmails,
            subject,
            text,
            html,
        });

        // Additionally, check if the delay option is set correctly
        const options = emailQueue.add.mock.calls[0][1];
        expect(options).toEqual({ delay });
    });

    it("enqueues an email without the html parameter", async () => {
        const testEmails = ["user@example.com"];
        const subject = "No HTML Subject";
        const text = "Email body without HTML";

        sendMail(testEmails, subject, text);

        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: testEmails,
            subject,
            text,
            html: undefined, // html should not be set if it's an empty string
        });

        // Check that no delay is set by default
        const options = emailQueue.add.mock.calls[0][1];
        expect(options).toEqual({ delay: 0 });
    });

    it("enqueues an email with a delay", async () => {
        const delay = 10000; // 10 seconds
        sendMail(["delayed@example.com"], "Delayed Email", "This is a delayed email.", "", delay);

        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: ["delayed@example.com"],
            subject: "Delayed Email",
            text: "This is a delayed email.",
            html: undefined,
        });

        // Check if the delay option is set correctly
        const options = emailQueue.add.mock.calls[0][1];
        expect(options).toEqual({ delay });
    });

    it("throws an error if no recipient is provided", async () => {
        expect(() => {
            sendMail([], "Subject", "Text body");
        }).toThrow();
    });
});

describe("Email task enqueuing tests", () => {
    let emailQueue;
    const originalSiteEmailUsername = process.env.SITE_EMAIL_USERNAME;
    const testEmail = "valid.email@example.com";

    beforeAll(async () => {
        emailQueue = new Bull("email");
        await setupEmailQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
        process.env.SITE_EMAIL_USERNAME = "admin@example.com";
    });

    afterEach(() => {
        process.env.SITE_EMAIL_USERNAME = originalSiteEmailUsername;
    });

    it("correctly enqueues email for sendResetPasswordLink function", async () => {
        const userId = "userId1";
        const code = "code1";
        sendResetPasswordLink(testEmail, userId, code);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
        // Make sure that the text and html both include `${userId1}:${code1}`
        const actualData = emailQueue.add.mock.calls[0][0];
        const partialLink = `${userId}:${code}`;
        expect(actualData.text).toContain(partialLink);
        expect(actualData.html).toContain(partialLink);
    });

    it("correctly enqueues email for sendVerificationLink function", async () => {
        const userId = "userId1";
        const code = "code1";
        sendVerificationLink(testEmail, userId, code);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
        // Make sure that the text and html both include `${userId1}:${code1}`
        const actualData = emailQueue.add.mock.calls[0][0];
        const partialLink = `${userId}:${code}`;
        expect(actualData.text).toContain(partialLink);
        expect(actualData.html).toContain(partialLink);
    });

    it("correctly enqueues feedback notification email for the admin", async () => {
        const feedbackText = "This is a feedback message.";
        const feedbackFrom = "user@example.com";
        feedbackNotifyAdmin(feedbackText, feedbackFrom);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [process.env.SITE_EMAIL_USERNAME],
            subject: "Received Vrooli Feedback!",
            text: `Feedback from ${feedbackFrom}: ${feedbackText}`,
        });
    });

    it("correctly handles anonymous feedback notification email for the admin", async () => {
        const feedbackText = "This is anonymous feedback.";
        feedbackNotifyAdmin(feedbackText);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [process.env.SITE_EMAIL_USERNAME],
            subject: "Received Vrooli Feedback!",
            text: `Feedback from anonymous: ${feedbackText}`,
        });
    });

    it("correctly enqueues email for sendPaymentThankYou function", async () => {
        sendPaymentThankYou(testEmail, PaymentType.Donation);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
    });

    it("correctly enqueues email for sendPaymentFailed function", async () => {
        sendPaymentFailed(testEmail, PaymentType.Donation);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
    });

    it("correctly enqueues email for sendCreditCardExpiringSoon function", async () => {
        sendCreditCardExpiringSoon(testEmail);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
    });

    it("correctly enqueues email for sendSubscriptionCanceled function", async () => {
        sendSubscriptionCanceled(testEmail);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
    });

    it("correctly enqueues email for sendSubscriptionEnded function", async () => {
        sendSubscriptionEnded(testEmail);
        await expectEmailToBeEnqueuedWith(emailQueue, {
            to: [testEmail],
        });
    });
});
