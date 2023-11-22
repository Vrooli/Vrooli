import { PubSub } from "./pubsub";

describe("PubSub", () => {
    let pubSub;

    beforeEach(() => {
        pubSub = PubSub.get();
    });

    test("should always return the same instance", () => {
        const anotherInstance = PubSub.get();
        expect(pubSub).toBe(anotherInstance);
    });

    test("should allow subscription to events", () => {
        const mockSubscriber = jest.fn();
        const token = pubSub.subscribe("snack", mockSubscriber);

        expect(typeof token).toBe("symbol");
        pubSub.publish("snack", { message: "Test Snack" });
        expect(mockSubscriber).toHaveBeenCalledWith({ message: "Test Snack" });
    });

    test("should allow unsubscription from events", () => {
        const mockSubscriber = jest.fn();
        const token = pubSub.subscribe("snack", mockSubscriber);

        pubSub.unsubscribe(token);
        pubSub.publish("snack", { message: "Test Snack" });
        expect(mockSubscriber).not.toHaveBeenCalled();
    });

    test("should notify subscribers with provided data", () => {
        const mockSubscriber = jest.fn();
        const testEventData = { message: "Test Event Data" };
        pubSub.subscribe("snack", mockSubscriber);

        pubSub.publish("snack", testEventData);
        expect(mockSubscriber).toHaveBeenCalledWith(testEventData);
    });

    test("should use default payload if no data is provided", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("fastUpdate", mockSubscriber);

        // Note: Assuming fastUpdate has a default payload as defined in defaultPayloads
        const defaultFastUpdatePayload = { on: true, duration: 1000 };
        pubSub.publish("fastUpdate");

        expect(mockSubscriber).toHaveBeenCalledWith(defaultFastUpdatePayload);
    });

    test("should not notify subscribers with default data if it is not defined", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("theme", mockSubscriber);

        // Note: Assuming theme does not have a default payload
        pubSub.publish("theme");

        // Depending on the implementation, you may expect the subscriber to be called with undefined or not called at all.
        // This test needs to be adjusted based on the specific behavior of your implementation.
        expect(mockSubscriber).toHaveBeenCalledWith(undefined); // or expect(mockSubscriber).not.toHaveBeenCalled();
    });
});
