import { generatePK, type Session, type SessionUser } from "@vrooli/shared";
import { describe, it, expect, afterAll, beforeEach, vi } from "vitest";
import { checkIfLoggedIn, getCurrentUser } from "./session.js";

describe("getCurrentUser", () => {
    it("should return an empty object if session is null", () => {
        expect(getCurrentUser(null)).toEqual({});
    });

    it("should return an empty object if session is undefined", () => {
        expect(getCurrentUser(undefined)).toEqual({});
    });

    it("should return an empty object if user is not logged in", () => {
        const session = {
            isLoggedIn: false,
            users: [],
        } as unknown as Session;
        expect(getCurrentUser(session)).toEqual({});
    });

    it("should return an empty object if users array is empty", () => {
        const session = {
            isLoggedIn: true,
            users: [],
        } as unknown as Session;
        expect(getCurrentUser(session)).toEqual({});
    });

    it("should return an empty object if users array is not an array", () => {
        const session = {
            isLoggedIn: true,
            users: null,
        } as Session;
        expect(getCurrentUser(session)).toEqual({});
    });

    it("should return an empty object if user ID is not a valid snowflake ID", () => {
        const session = {
            isLoggedIn: true,
            users: [{ id: "invalid-id" }] as SessionUser[],
        } as Session;
        expect(getCurrentUser(session)).toEqual({});
    });

    it("should return user data if user ID is a valid snowflake ID", () => {
        const validId = generatePK().toString();
        const expectedUser = {
            id: validId,
            languages: ["en", "fr"],
            name: "Test User",
            theme: "dark",
        } as SessionUser;
        const session = {
            isLoggedIn: true,
            users: [expectedUser],
        } as Session;
        expect(getCurrentUser(session)).toEqual(expectedUser);
    });
});

describe("checkIfLoggedIn", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.localStorage.clear();
    });
    afterAll(() => {
        global.localStorage.clear();
        vi.restoreAllMocks();
    });

    it("should return true if session is null and local storage has isLoggedIn set to true", () => {
        localStorage.setItem("isLoggedIn", "true");
        expect(checkIfLoggedIn(null)).toEqual(true);
    });

    it("should return false if session is null and local storage does not have isLoggedIn set to true", () => {
        localStorage.setItem("isLoggedIn", "false");
        expect(checkIfLoggedIn(null)).toEqual(false);
    });

    it("should return false is session is valid but isLoggedIn is false", () => {
        const session = {
            isLoggedIn: false,
        } as Session;
        expect(checkIfLoggedIn(session)).toEqual(false);
    });

    it("should return true is session is true and local storage has an invalid value, since the session overrides local storage", () => {
        const session = {
            isLoggedIn: true,
        } as Session;
        localStorage.setItem("isLoggedIn", "chicken");
        expect(checkIfLoggedIn(session)).toEqual(true);
    });

    it("should return false is session is false and local storage is invalid", () => {
        const session = {
            isLoggedIn: false,
        } as Session;
        localStorage.setItem("isLoggedIn", "chicken");
        expect(checkIfLoggedIn(session)).toEqual(false);
    });

    it("should return true is session is valid and isLoggedIn is true", () => {
        const session = {
            isLoggedIn: true,
        } as Session;
        expect(checkIfLoggedIn(session)).toEqual(true);
    });
});
