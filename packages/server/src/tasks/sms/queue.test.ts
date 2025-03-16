import { expect } from "chai";
import * as yup from "yup";
import Bull from "../../__mocks__/bull.js";
import { sendSms, sendSmsVerification, setupSmsQueue } from "./queue.js";

/**
 * Validates the SMS task object structure.
 */
const smsSchema = yup.object().shape({
    to: yup.array().of(yup.string().matches(/^\+[1-9]\d{1,14}$/, "Must be a valid international phone number")).min(1, "The \"to\" field must contain at least one phone number"),
    body: yup.string().required("Body is required").min(1, "Body cannot be empty"),
});

/**
 * Helper function to assert SMS task properties.
 * @param mockQueue The mocked SMS queue.
 * @param expectedData The expected data object to validate against.
 */
async function expectSmsToBeEnqueuedWith(mockQueue, expectedData) {
    expect(mockQueue.add).toHaveBeenCalled();
    const actualData = mockQueue.add.mock.calls[0][0];
    await expect(smsSchema.validate(actualData)).resolves.to.deep.equal(actualData);
    if (expectedData) {
        expect(actualData).toMatchObject(expectedData);
    }
}

describe("sendSms function tests", () => {
    let smsQueue;

    before(async () => {
        smsQueue = new Bull("sms");
        await setupSmsQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
    });

    it("enqueues an SMS with valid parameters", async () => {
        const recipients = ["+1234567890"];
        const body = "Test SMS body";

        sendSms(recipients, body);

        await expectSmsToBeEnqueuedWith(smsQueue, {
            to: recipients,
            body,
        });
    });

    it("throws an error if no recipient is provided", () => {
        const body = "Test SMS body";
        expect(() => {
            sendSms([], body);
        }).to.throw();
    });
});

describe("sendSmsVerification function tests", () => {
    let smsQueue;

    before(async () => {
        smsQueue = new Bull("sms");
        await setupSmsQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
    });

    it("enqueues a verification code SMS", async () => {
        const phoneNumber = "+1234567890";
        const code = "123456";

        sendSmsVerification(phoneNumber, code);

        await expectSmsToBeEnqueuedWith(smsQueue, {
            to: [phoneNumber],
        });
    });
});
