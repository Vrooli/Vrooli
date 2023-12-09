import { Session, uuid } from "@local/shared";
import { getUserLanguages } from "./translationTools";

// Mocks for navigator.language
const mockNavigatorLanguage = (language) => {
    Object.defineProperty(global.navigator, "language", {
        value: language,
        writable: true,
    });
};

// Utility function for creating a session object
const createSession = (languages: string[] | null | undefined) => ({
    __typename: "Session",
    isLoggedIn: true,
    users: [{
        __typename: "SessionUser",
        id: uuid(),
        languages,
    }],
} as unknown as Session);

describe("getUserLanguages", () => {
    // Reset navigator.language before each test
    beforeEach(() => {
        mockNavigatorLanguage(undefined);
    });

    it("should return at least English when useDefault is true and no languages are found", () => {
        expect(getUserLanguages(null, true)).toEqual(["en"]);
        expect(getUserLanguages(undefined)).toEqual(["en"]);
    });

    it("should return user defined languages from session", () => {
        const session = createSession(["en-US", "fr"]);
        expect(getUserLanguages(session)).toEqual(["en", "fr"]);
    });

    it("should handle session with empty language array", () => {
        const session = createSession([]);
        expect(getUserLanguages(session)).toEqual(["en"]);
    });

    it("should handle session with missing languages property", () => {
        const session = createSession(undefined);
        expect(getUserLanguages(session)).toEqual(["en"]);
    });

    it("should return navigator language if no session languages", () => {
        mockNavigatorLanguage("de");
        const session = createSession([]);
        expect(getUserLanguages(session, false)).toEqual(["de"]);
    });

    it("should return empty array when useDefault is false and no languages are found", () => {
        expect(getUserLanguages(null, false)).toEqual([]);
        expect(getUserLanguages(undefined, false)).toEqual([]);
    });

    it("should handle cases where navigator does not return a language", () => {
        mockNavigatorLanguage(undefined);
        const session = createSession(undefined);
        expect(getUserLanguages(session, false)).toEqual([]);
    });

    // Additional cases can be added as needed
});
