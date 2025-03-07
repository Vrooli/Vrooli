import { expect } from "chai";
import { PubSub } from "./pubsub.js";

describe("PubSub", () => {
    let pubSub: PubSub;

    beforeEach(() => {
        pubSub = PubSub.get();
        // Clear the subscribers map before each test to prevent interference between tests
        (pubSub as any).subscribers = new Map();
    });

    it("should always return the same instance", () => {
        const anotherInstance = PubSub.get();
        expect(pubSub).to.equal(anotherInstance);
    });

    it("should allow subscription to events", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("snack", mockSubscriber);

        const testEventData = { message: "Test Snack", severity: "Success" } as const;
        pubSub.publish("snack", testEventData);
        expect(mockSubscriber).toHaveBeenCalledWith(testEventData);
    });

    it("should allow unsubscription from events", () => {
        const mockSubscriber = jest.fn();
        const unsubscribe = pubSub.subscribe("snack", mockSubscriber);

        unsubscribe();
        pubSub.publish("snack", { message: "Test Snack", severity: "Success" });
        expect(mockSubscriber).not.toHaveBeenCalled();
    });

    it("should notify subscribers with provided data", () => {
        const mockSubscriber = jest.fn();
        const testEventData = { message: "Test Event Data", severity: "Error" } as const;
        pubSub.subscribe("snack", mockSubscriber);

        pubSub.publish("snack", testEventData);
        expect(mockSubscriber).toHaveBeenCalledWith(testEventData);
    });

    // it("should use default payload if no data is provided", () => {
    //     const mockSubscriber = jest.fn();
    //     pubSub.subscribe("fastUpdate", mockSubscriber);

    //     // Adjust this to match your actual default payload for "fastUpdate"
    //     const defaultFastUpdatePayload = { on: true, duration: 1000 };
    //     pubSub.publish("fastUpdate");

    //     expect(mockSubscriber).toHaveBeenCalledWith(defaultFastUpdatePayload);
    // });

    it("should notify subscribers with undefined if default payload is not defined", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("theme", mockSubscriber);

        pubSub.publish("theme");

        expect(mockSubscriber).toHaveBeenCalledWith(undefined);
    });

    // Tests for hasSubscribers

    it("hasSubscribers should return true when there are subscribers for an event type", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("snack", mockSubscriber);

        expect(pubSub.hasSubscribers("snack")).to.equal(true);
    });

    it("hasSubscribers should return false when there are no subscribers for an event type", () => {
        expect(pubSub.hasSubscribers("snack")).to.equal(false);
    });

    it("hasSubscribers should return true when a subscriber matches the filter function", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("snack", mockSubscriber, { componentType: "form" });

        const hasFormSubscribers = pubSub.hasSubscribers("snack", metadata => metadata?.componentType === "form");
        expect(hasFormSubscribers).to.equal(true);
    });

    it("hasSubscribers should return false when no subscribers match the filter function", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("snack", mockSubscriber, { componentType: "chat" });

        const hasFormSubscribers = pubSub.hasSubscribers("snack", metadata => metadata?.componentType === "form");
        expect(hasFormSubscribers).to.equal(false);
    });

    it("hasSubscribers should handle subscribers with undefined metadata", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("snack", mockSubscriber); // No metadata provided

        const hasSubscribers = pubSub.hasSubscribers("snack");
        expect(hasSubscribers).to.equal(true);

        const hasFormSubscribers = pubSub.hasSubscribers("snack", metadata => metadata?.componentType === "form");
        expect(hasFormSubscribers).to.equal(false);
    });

    it("hasSubscribers should return false after all subscribers have unsubscribed", () => {
        const mockSubscriber = jest.fn();
        const unsubscribe = pubSub.subscribe("snack", mockSubscriber);

        unsubscribe();

        expect(pubSub.hasSubscribers("snack")).to.equal(false);
    });

    // Tests for publish with filter function

    it("publish should send event to all subscribers when no filter function is provided", () => {
        const mockSubscriber1 = jest.fn();
        const mockSubscriber2 = jest.fn();

        pubSub.subscribe("snack", mockSubscriber1, { componentType: "form" });
        pubSub.subscribe("snack", mockSubscriber2, { componentType: "chat" });

        const eventData = { message: "Broadcast Event", severity: "Warning" } as const;
        pubSub.publish("snack", eventData);

        expect(mockSubscriber1).toHaveBeenCalledWith(eventData);
        expect(mockSubscriber2).toHaveBeenCalledWith(eventData);
    });

    it("publish should send event only to subscribers matching the filter function", () => {
        const mockSubscriber1 = jest.fn();
        const mockSubscriber2 = jest.fn();
        const mockSubscriber3 = jest.fn();

        pubSub.subscribe("snack", mockSubscriber1, { componentType: "form" });
        pubSub.subscribe("snack", mockSubscriber2, { componentType: "chat" });
        pubSub.subscribe("snack", mockSubscriber3, { componentType: "form" });

        const eventData = { message: "Filtered Event", severity: "Error" } as const;
        pubSub.publish("snack", eventData, metadata => metadata?.componentType === "form");

        expect(mockSubscriber1).toHaveBeenCalledWith(eventData);
        expect(mockSubscriber2).not.toHaveBeenCalled();
        expect(mockSubscriber3).toHaveBeenCalledWith(eventData);
    });

    it("publish should not send event to any subscribers when filter function matches none", () => {
        const mockSubscriber1 = jest.fn();
        const mockSubscriber2 = jest.fn();

        pubSub.subscribe("snack", mockSubscriber1, { componentType: "form" });
        pubSub.subscribe("snack", mockSubscriber2, { componentType: "chat" });

        const eventData = { message: "No Match Event", severity: "Success" } as const;
        pubSub.publish("snack", eventData, metadata => metadata?.componentType === "other");

        expect(mockSubscriber1).not.toHaveBeenCalled();
        expect(mockSubscriber2).not.toHaveBeenCalled();
    });

    it("publish should handle subscribers with undefined metadata", () => {
        const mockSubscriber1 = jest.fn();
        const mockSubscriber2 = jest.fn();

        pubSub.subscribe("snack", mockSubscriber1); // No metadata
        pubSub.subscribe("snack", mockSubscriber2, { componentType: "form" });

        const eventData = { message: "Event with Filter", severity: "Success" } as const;
        pubSub.publish("snack", eventData, metadata => metadata?.componentType === "form");

        expect(mockSubscriber1).not.toHaveBeenCalled();
        expect(mockSubscriber2).toHaveBeenCalledWith(eventData);
    });

    it("publish should send event to all subscribers when filter function is not provided", () => {
        const mockSubscriber = jest.fn();
        pubSub.subscribe("snack", mockSubscriber);

        const eventData = { message: "Event without Filter", severity: "Success" } as const;
        pubSub.publish("snack", eventData);

        expect(mockSubscriber).toHaveBeenCalledWith(eventData);
    });

    it("publish should not fail when there are no subscribers", () => {
        expect(() => {
            pubSub.publish("snack", { message: "No Subscribers", severity: "Success" });
        }).not.to.throw();
    });
});
