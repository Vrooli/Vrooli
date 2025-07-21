import { expect, describe, it } from "vitest";
import { connectOwner } from "./connectOwner.js";
import { type SessionUser } from "@vrooli/shared";

describe("connectOwner", () => {
    const mockSession: SessionUser = {
        id: "user123",
        languages: ["en"],
    };

    describe("team connection", () => {
        it("connects to team when teamConnect is specified", () => {
            const input = {
                teamConnect: "team456",
                userConnect: null,
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByTeam: { connect: { id: "team456" } },
            });
        });

        it("prioritizes team over user when both are specified", () => {
            const input = {
                teamConnect: "team456",
                userConnect: "user789",
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByTeam: { connect: { id: "team456" } },
            });
        });

        it("connects to team with empty string teamConnect", () => {
            const input = {
                teamConnect: "", // Empty string should still be processed
                userConnect: null,
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByTeam: { connect: { id: "" } },
            });
        });
    });

    describe("user connection", () => {
        it("connects to specified user when userConnect is provided", () => {
            const input = {
                teamConnect: null,
                userConnect: "user789",
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user789" } },
            });
        });

        it("connects to specified user with undefined teamConnect", () => {
            const input = {
                teamConnect: undefined,
                userConnect: "user789",
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user789" } },
            });
        });

        it("connects to user with empty string userConnect", () => {
            const input = {
                teamConnect: null,
                userConnect: "", // Empty string should still be processed
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "" } },
            });
        });
    });

    describe("default behavior", () => {
        it("connects to session user when neither is specified", () => {
            const input = {
                teamConnect: null,
                userConnect: null,
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user123" } },
            });
        });

        it("connects to session user with undefined values", () => {
            const input = {
                teamConnect: undefined,
                userConnect: undefined,
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user123" } },
            });
        });

        it("connects to session user with empty object", () => {
            const input = {};

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user123" } },
            });
        });

        it("connects to session user when both are falsy but not null/undefined", () => {
            const input = {
                teamConnect: null,
                userConnect: null,
                // Other properties should be preserved in the input type
                otherProp: "value",
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user123" } },
            });
        });
    });

    describe("edge cases", () => {
        it("handles session with additional properties", () => {
            const extendedSession: SessionUser = {
                id: "user999",
                languages: ["en", "es"],
                // Additional properties that might exist
            };

            const input = {
                teamConnect: null,
                userConnect: null,
            };

            const result = connectOwner(input, extendedSession);

            expect(result).toEqual({
                ownedByUser: { connect: { id: "user999" } },
            });
        });

        it("preserves input type while extracting connect fields", () => {
            interface CustomInput {
                teamConnect?: string | null;
                userConnect?: string | null;
                customField: string;
                anotherField: number;
            }

            const input: CustomInput = {
                teamConnect: "team123",
                userConnect: null,
                customField: "test",
                anotherField: 42,
            };

            const result = connectOwner(input, mockSession);

            expect(result).toEqual({
                ownedByTeam: { connect: { id: "team123" } },
            });
            // Original input should remain unchanged
            expect(input.customField).toBe("test");
            expect(input.anotherField).toBe(42);
        });

        it("handles all falsy values correctly", () => {
            // Test null
            expect(connectOwner({ teamConnect: null, userConnect: null }, mockSession))
                .toEqual({ ownedByUser: { connect: { id: "user123" } } });

            // Test undefined
            expect(connectOwner({ teamConnect: undefined, userConnect: undefined }, mockSession))
                .toEqual({ ownedByUser: { connect: { id: "user123" } } });

            // Test empty string (should connect to empty string, not default)
            expect(connectOwner({ teamConnect: "", userConnect: null }, mockSession))
                .toEqual({ ownedByTeam: { connect: { id: "" } } });

            // Test zero as string (should connect)
            expect(connectOwner({ teamConnect: "0", userConnect: null }, mockSession))
                .toEqual({ ownedByTeam: { connect: { id: "0" } } });
        });
    });
});
