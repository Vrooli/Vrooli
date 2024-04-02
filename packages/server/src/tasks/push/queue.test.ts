import * as yup from "yup";
import Bull from "../../__mocks__/bull";
import { sendPush, setupPushQueue } from "./queue";

/**
 * Validates the push notification task object structure.
 */
const pushPayloadSchema = yup.object().shape({
    endpoint: yup.string().required("Endpoint is required"),
    keys: yup.object().shape({
        p256dh: yup.string().required("p256dh key is required"),
        auth: yup.string().required("Auth key is required"),
    }),
    body: yup.string().required("Body is required"),
    icon: yup.string().url().nullable().notRequired(),
    link: yup.string().url().nullable().notRequired(),
    title: yup.string().nullable().notRequired(),
});

/**
 * Helper function to assert push notification task properties.
 * @param mockQueue The mocked push queue.
 * @param expectedData The expected data object to validate against.
 * @param expectedOptions The expected options object to validate against.
 */
const expectPushToBeEnqueuedWith = async (mockQueue, expectedData, expectedOptions = { delay: 0 }) => {
    expect(mockQueue.add).toHaveBeenCalled();
    const actualData = mockQueue.add.mock.calls[0][0];
    const actualOptions = mockQueue.add.mock.calls[0][1];
    await expect(pushPayloadSchema.validate(actualData)).resolves.toEqual(expectedData);
    expect(actualOptions).toMatchObject(expectedOptions);
};

describe("sendPush function tests", () => {
    let pushQueue;

    beforeAll(async () => {
        pushQueue = new Bull("push");
        await setupPushQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
    });

    it("enqueues a push notification with all parameters and default delay", async () => {
        const subscription = {
            endpoint: "https://example.com/push",
            keys: {
                p256dh: "testP256dh",
                auth: "testAuth",
            },
        };
        const payload = {
            body: "Test Push Body",
            icon: "https://example.com/icon.png",
            link: "https://example.com",
            title: "Test Push Title",
        };

        sendPush(subscription, payload);

        await expectPushToBeEnqueuedWith(pushQueue, {
            ...subscription,
            ...payload,
        });
    });

    it("enqueues a push notification with a custom delay", async () => {
        const subscription = {
            endpoint: "https://example.com/push",
            keys: {
                p256dh: "testP256dh",
                auth: "testAuth",
            },
        };
        const payload = {
            body: "Delayed Push Body",
            icon: "https://example.com/icon-delayed.png",
            link: "https://example.com/delay",
            title: "Delayed Push Title",
        };
        const delay = 5000; // 5 seconds

        sendPush(subscription, payload, delay);

        await expectPushToBeEnqueuedWith(pushQueue, {
            ...subscription,
            ...payload,
        }, { delay });
    });
});
