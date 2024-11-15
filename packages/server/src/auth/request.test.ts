/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DEFAULT_LANGUAGE, SECONDS_1_MS } from "@local/shared";
import { generateKeyPairSync } from "crypto";
import jwt, { JwtPayload } from "jsonwebtoken";
import { isJwtExpired, parseAcceptLanguage, verifyJwt } from "./request";

describe("parseAcceptLanguage", () => {
    it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is missing`, () => {
        const req = { headers: {} };
        expect(parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
    });

    it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is "*" `, () => {
        const req = { headers: { "accept-language": "*" } };
        expect(parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
    });

    it("should parse \"en-US,en;q=0.9,fr;q=0.8\" correctly", () => {
        const req = { headers: { "accept-language": "en-US,en;q=0.9,fr;q=0.8" } };
        expect(parseAcceptLanguage(req)).toEqual(["en", "en", "fr"]);
    });

    it("should parse \"fr-CA,fr-FR;q=0.8,en-US;q=0.5,en;q=0.3\" correctly", () => {
        const req = { headers: { "accept-language": "fr-CA,fr-FR;q=0.8,en-US;q=0.5,en;q=0.3" } };
        expect(parseAcceptLanguage(req)).toEqual(["fr", "fr", "en", "en"]);
    });

    it("should parse \"es\" correctly", () => {
        const req = { headers: { "accept-language": "es" } };
        expect(parseAcceptLanguage(req)).toEqual(["es"]);
    });

    it("should parse \"zh-Hant,zh-Hans;q=0.9,ja;q=0.8\" correctly", () => {
        const req = { headers: { "accept-language": "zh-Hant,zh-Hans;q=0.9,ja;q=0.8" } };
        expect(parseAcceptLanguage(req)).toEqual(["zh", "zh", "ja"]);
    });

    it("should parse \"en-GB,en;q=0.8\" correctly", () => {
        const req = { headers: { "accept-language": "en-GB,en;q=0.8" } };
        expect(parseAcceptLanguage(req)).toEqual(["en", "en"]);
    });

    it("should handle accept-language with wildcard and language codes", () => {
        const req = { headers: { "accept-language": "*,en;q=0.5" } };
        expect(parseAcceptLanguage(req)).toEqual(["*", "en"]);
    });

    it("should handle invalid accept-language values", () => {
        const req = { headers: { "accept-language": "invalid" } };
        expect(parseAcceptLanguage(req)).toEqual(["invalid"]);
    });

    it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is empty string`, () => {
        const req = { headers: { "accept-language": "" } };
        expect(parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
    });

    it("should handle accept-language with extra spaces", () => {
        const req = { headers: { "accept-language": " en-US , en ;q=0.9 " } };
        expect(parseAcceptLanguage(req)).toEqual([" en", " en "]);
    });

    it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is null`, () => {
        const req = { headers: { "accept-language": null } };
        expect(parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
    });

    it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is undefined`, () => {
        const req = { headers: { "accept-language": undefined } };
        expect(parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
    });

    it("should handle accept-language with uppercase letters", () => {
        const req = { headers: { "accept-language": "EN-US,en;q=0.8" } };
        expect(parseAcceptLanguage(req)).toEqual(["EN", "en"]);
    });

    it(`should return ["${DEFAULT_LANGUAGE}"]  when accept-language header is not a string`, () => {
        const req = { headers: { "accept-language": 42 } };
        expect(parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
    });
});

describe("isJwtExpired", () => {
    // Fixed current time for consistent testing (e.g., Jan 1, 2021 00:00:00 GMT)
    const fixedTime = 1609459200 * SECONDS_1_MS; // in milliseconds

    beforeEach(() => {
        jest.spyOn(Date, "now").mockReturnValue(fixedTime);
    });

    afterEach(() => {
        jest.restoreAllMocks();
    });

    test("returns true when exp is undefined", () => {
        const payload: JwtPayload = { foo: "bar" };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns true when exp is NaN", () => {
        const payload: JwtPayload = { exp: NaN };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns true when exp is Infinity", () => {
        const payload: JwtPayload = { exp: Infinity };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns true when exp is -Infinity", () => {
        const payload: JwtPayload = { exp: -Infinity };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns true when exp is less than current time", () => {
        const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) - 1 };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns false when exp is exactly equal to current time", () => {
        const payload: JwtPayload = { exp: fixedTime / SECONDS_1_MS };
        expect(isJwtExpired(payload)).toBe(false);
    });

    test("returns false when exp is greater than current time", () => {
        const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) + 1 };
        expect(isJwtExpired(payload)).toBe(false);
    });

    test("handles payload with additional properties correctly", () => {
        const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) + 1, foo: "bar", baz: 123 };
        expect(isJwtExpired(payload)).toBe(false);
    });

    test("handles payload with exp as zero (expired)", () => {
        const payload: JwtPayload = { exp: 0 };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("handles payload with negative exp (expired)", () => {
        const payload: JwtPayload = { exp: -100 };
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns true when payload is null", () => {
        const payload = null;
        // @ts-ignore Testing runtime scenario
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("returns true when payload is undefined", () => {
        const payload = undefined;
        // @ts-ignore Testing runtime scenario
        expect(isJwtExpired(payload)).toBe(true);
    });

    test("handles non-integer exp values correctly", () => {
        const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) + 123.456 };
        expect(isJwtExpired(payload)).toBe(false);
    });
});

describe("verifyJwt", () => {
    let privateKey: string;
    let publicKey: string;

    beforeAll(() => {
        // Generate a 2048-bit RSA key pair
        const { privateKey: privKey, publicKey: pubKey } = generateKeyPairSync("rsa", {
            modulusLength: 2048,
            publicKeyEncoding: {
                type: "spki",
                format: "pem",
            },
            privateKeyEncoding: {
                type: "pkcs8",
                format: "pem",
            },
        });

        privateKey = privKey;
        publicKey = pubKey;
    });

    test("should resolve with payload for a valid token", async () => {
        const payload = { userId: "12345", role: "admin" };
        const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1h" });

        await expect(verifyJwt(token, publicKey)).resolves.toMatchObject(payload);
    });

    test("should reject if token is invalid", async () => {
        const invalidToken = "invalid.token.here";

        await expect(verifyJwt(invalidToken, publicKey)).rejects.toThrow(jwt.JsonWebTokenError);
    });

    describe("expired", () => {
        describe("reject", () => {
            test("negative expiresIn", async () => {
                const payload = { userId: "12345", role: "admin" };
                const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "-10s" });

                await expect(verifyJwt(token, publicKey)).rejects.toThrow(jwt.TokenExpiredError);
            });
            test("negative exp", async () => {
                const payload = { userId: "12345", role: "admin", exp: -1 };
                const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                await expect(verifyJwt(token, publicKey)).rejects.toThrow(jwt.TokenExpiredError);
            });
            test("zero exp", async () => {
                const payload = { userId: "12345", role: "admin", exp: 0 };
                const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                await expect(verifyJwt(token, publicKey)).rejects.toThrow(jwt.TokenExpiredError);
            });
            test("exp less than current time", async () => {
                const payload = { userId: "12345", role: "admin", exp: (Date.now() / SECONDS_1_MS) - 1 };
                const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                await expect(verifyJwt(token, publicKey)).rejects.toThrow(jwt.TokenExpiredError);
            });
        });
        describe("resolve", () => {
            test("positive expiresIn", async () => {
                const payload = { userId: "12345", role: "admin" };
                const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1s" });

                await expect(verifyJwt(token, publicKey)).resolves.toMatchObject(payload);
            });
            test("exp greater than current time", async () => {
                const payload = { userId: "12345", role: "admin", exp: (Date.now() / SECONDS_1_MS) + 1 };
                const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                await expect(verifyJwt(token, publicKey)).resolves.toMatchObject(payload);
            });
        });
    });

    test("should reject if public key is incorrect", async () => {
        // Generate a different key pair to simulate an incorrect public key
        const { publicKey: incorrectPublicKey } = generateKeyPairSync("rsa", {
            modulusLength: 2048,
            publicKeyEncoding: {
                type: "spki",
                format: "pem",
            },
            privateKeyEncoding: {
                type: "pkcs8",
                format: "pem",
            },
        });

        const payload = { userId: "12345", role: "admin" };
        const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1h" });

        await expect(verifyJwt(token, incorrectPublicKey)).rejects.toThrow(jwt.JsonWebTokenError);
    });

    test("should reject if token signed with different algorithm", async () => {
        const hmacKey = "mysecret";
        const payload = { userId: "12345", role: "admin" };
        const token = jwt.sign(payload, hmacKey, { algorithm: "HS256", expiresIn: "1h" });

        await expect(verifyJwt(token, publicKey)).rejects.toThrow(jwt.JsonWebTokenError);
    });

    test("should reject if token is missing", async () => {
        const token = "";

        await expect(verifyJwt(token, publicKey)).rejects.toThrow(jwt.JsonWebTokenError);
    });

    test("should reject if token is undefined", async () => {
        const token = undefined as unknown as string;

        await expect(verifyJwt(token, publicKey)).rejects.toThrow(Error);
    });

    test("should reject with error if jwt.verify returns undefined payload", async () => {
        const payload = { userId: "12345", role: "admin" };
        const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1h" });

        const verifySpy = jest.spyOn(jwt, "verify").mockImplementation((token, publicKey, options, callback) => {
            (callback as jwt.VerifyCallback)(null, undefined);
        });

        await expect(verifyJwt(token, publicKey)).rejects.toThrow(expect.any(Error));

        verifySpy.mockRestore();
    });
});
