import { describe, expect, it } from "vitest";
import { EventTypes, extractEventData, isUnifiedEvent, type UnifiedEvent } from "./socketEvents.js";

describe("UnifiedEvent helpers", () => {
    describe("isUnifiedEvent", () => {
        it("should return true for valid UnifiedEvent", () => {
            const validEvent: UnifiedEvent<{ test: string }> = {
                id: "test-id-123",
                type: EventTypes.CHAT.MESSAGE_ADDED,
                timestamp: new Date(),
                data: { test: "value" },
            };
            
            expect(isUnifiedEvent(validEvent)).toBe(true);
        });

        it("should return false for plain payload", () => {
            const plainPayload = {
                chatId: "123",
                messages: [],
            };
            
            expect(isUnifiedEvent(plainPayload)).toBe(false);
        });

        it("should return false for null", () => {
            expect(isUnifiedEvent(null)).toBe(false);
        });

        it("should return false for undefined", () => {
            expect(isUnifiedEvent(undefined)).toBe(false);
        });

        it("should return false for non-object values", () => {
            expect(isUnifiedEvent("string")).toBe(false);
            expect(isUnifiedEvent(123)).toBe(false);
            expect(isUnifiedEvent(true)).toBe(false);
        });

        it("should return false for objects missing required properties", () => {
            expect(isUnifiedEvent({ id: "123" })).toBe(false);
            expect(isUnifiedEvent({ id: "123", type: "test" })).toBe(false);
            expect(isUnifiedEvent({ id: "123", type: "test", timestamp: new Date() })).toBe(false);
            expect(isUnifiedEvent({ type: "test", timestamp: new Date(), data: {} })).toBe(false);
        });
    });

    describe("extractEventData", () => {
        it("should extract data from UnifiedEvent", () => {
            const data = { chatId: "123", messages: ["test"] };
            const unifiedEvent: UnifiedEvent<typeof data> = {
                id: "event-123",
                type: EventTypes.CHAT.MESSAGE_ADDED,
                timestamp: new Date(),
                data,
            };
            
            expect(extractEventData(unifiedEvent)).toEqual(data);
        });

        it("should return payload as-is if not UnifiedEvent", () => {
            const plainPayload = { chatId: "123", messages: ["test"] };
            
            expect(extractEventData(plainPayload)).toEqual(plainPayload);
        });

        it("should handle null values", () => {
            expect(extractEventData(null)).toBe(null);
        });

        it("should handle undefined values", () => {
            expect(extractEventData(undefined)).toBe(undefined);
        });

        it("should handle primitive values", () => {
            expect(extractEventData("string")).toBe("string");
            expect(extractEventData(123)).toBe(123);
            expect(extractEventData(true)).toBe(true);
        });
    });

    describe("UnifiedEvent wrapping and unwrapping flow", () => {
        it("should successfully wrap and unwrap event data", () => {
            const originalPayload = {
                chatId: "chat-123",
                messages: [
                    { id: "msg-1", content: "Hello", userId: "user-1" },
                ],
            };

            // Simulate server wrapping
            const wrappedEvent: UnifiedEvent<typeof originalPayload> = {
                id: "event-456",
                type: EventTypes.CHAT.MESSAGE_ADDED,
                timestamp: new Date(),
                data: originalPayload,
            };

            // Verify it's recognized as UnifiedEvent
            expect(isUnifiedEvent(wrappedEvent)).toBe(true);

            // Simulate client unwrapping
            const unwrappedData = extractEventData(wrappedEvent);
            
            // Verify data is preserved
            expect(unwrappedData).toEqual(originalPayload);
        });
    });
});
