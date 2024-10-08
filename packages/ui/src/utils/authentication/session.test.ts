import { Session, SessionUser, uuid } from "@local/shared";
import { checkIfLoggedIn, getCurrentUser } from "./session";

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

    it("should return an empty object if user ID is not a valid UUID", () => {
        const session = {
            isLoggedIn: true,
            users: [{ id: "invalid-uuid" }] as SessionUser[],
        } as Session;
        expect(getCurrentUser(session)).toEqual({});
    });

    it("should return user data if user ID is a valid UUID", () => {
        const validUUID = uuid();
        const expectedUser = {
            id: validUUID,
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
        jest.clearAllMocks();
        global.localStorage.clear();
    });
    afterAll(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    it("should return true if session is null and local storage has isLoggedIn set to true", () => {
        localStorage.setItem("isLoggedIn", "true");
        expect(checkIfLoggedIn(null)).toBe(true);
    });

    it("should return false if session is null and local storage does not have isLoggedIn set to true", () => {
        localStorage.setItem("isLoggedIn", "false");
        expect(checkIfLoggedIn(null)).toBe(false);
    });

    it("should return false is session is valid but isLoggedIn is false", () => {
        const session = {
            isLoggedIn: false,
        } as Session;
        expect(checkIfLoggedIn(session)).toBe(false);
    });

    it("should return true is session is true and local storage has an invalid value, since the session overrides local storage", () => {
        const session = {
            isLoggedIn: true,
        } as Session;
        localStorage.setItem("isLoggedIn", "chicken");
        expect(checkIfLoggedIn(session)).toBe(true);
    });

    it("should return false is session is false and local storage is invalid", () => {
        const session = {
            isLoggedIn: false,
        } as Session;
        localStorage.setItem("isLoggedIn", "chicken");
        expect(checkIfLoggedIn(session)).toBe(false);
    });

    it("should return true is session is valid and isLoggedIn is true", () => {
        const session = {
            isLoggedIn: true,
        } as Session;
        expect(checkIfLoggedIn(session)).toBe(true);
    });
});
