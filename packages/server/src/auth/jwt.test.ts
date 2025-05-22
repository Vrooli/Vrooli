/* eslint-disable @typescript-eslint/ban-ts-comment */
import { COOKIE, MINUTES_15_MS, SECONDS_1_MS } from "@local/shared";
import { expect } from "chai";
import { generateKeyPairSync } from "crypto";
import { type Response } from "express";
import jwt, { type JwtPayload } from "jsonwebtoken";
import { UI_URL_REMOTE } from "../server.js";
import { ACCESS_TOKEN_EXPIRATION_MS, JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "./jwt.js";

describe("JsonWebToken", () => {
    let originalNodeEnv: string | undefined;

    before(() => {
        originalNodeEnv = process.env.NODE_ENV;
    });
    afterEach(() => {
        process.env.NODE_ENV = originalNodeEnv;
    });

    describe("Singleton Behavior", () => {
        it("get() should return the same instance", () => {
            const instance1 = JsonWebToken.get();
            const instance2 = JsonWebToken.get();
            expect(instance1).to.equal(instance2);
        });
    });

    describe("getJwtCookieOptions", () => {
        it("returns correct options when NODE_ENV is 'production'", () => {
            process.env.NODE_ENV = "production";
            const options = JsonWebToken.getJwtCookieOptions();
            expect(options).to.deep.equal({
                httpOnly: true,
                sameSite: "lax",
                secure: true,
                maxAge: MINUTES_15_MS,
            });
        });

        it("returns correct options when NODE_ENV is not 'production'", () => {
            process.env.NODE_ENV = "development";
            const options = JsonWebToken.getJwtCookieOptions();
            expect(options).to.deep.equal({
                httpOnly: true,
                sameSite: "lax",
                secure: false,
                maxAge: MINUTES_15_MS,
            });
        });
    });

    describe("basicToken", () => {
        it("returns token with default iss and exp", () => {
            const instance = JsonWebToken.get();
            const token = instance.basicToken();
            expect(token.iss).to.equal(UI_URL_REMOTE);
            expect(token.iat * SECONDS_1_MS).to.be.at.most(Date.now());
            expect(Math.abs((token.exp * SECONDS_1_MS) - ACCESS_TOKEN_EXPIRATION_MS - Date.now())).to.be.at.most(SECONDS_1_MS);
        });

        it("returns token with custom iss", () => {
            const instance = JsonWebToken.get();
            const customIss = "http://custom-issuer.com";
            const token = instance.basicToken(customIss);
            expect(token.iss).to.equal(customIss);
        });

        it("returns token with custom exp", () => {
            const instance = JsonWebToken.get();
            const customExp = JsonWebToken.createExpirationTime(100_000);
            const token = instance.basicToken(undefined, customExp);
            expect(token.exp).to.equal(customExp);
        });
    });

    describe("calculateTokenSizes", () => {
        let instance: JsonWebToken;
        let privateKey: string;
        let publicKey: string;
        beforeEach(() => {
            instance = JsonWebToken.get();
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
            instance.setupTestEnvironment({ privateKey, publicKey });
        });

        it("calculates token sizes", () => {
            instance.calculateTokenSizes();
            expect(instance.getTokenHeaderSize()).to.be.greaterThan(0);
            expect(instance.getTokenSignatureSize()).to.be.greaterThan(0);
            expect(instance.getMaxPayloadSize()).to.be.greaterThan(0);
        });

        it("does not recalculate sizes if already calculated", () => {
            instance.calculateTokenSizes();
            const signSpy = jest.spyOn(instance, "sign");
            instance.calculateTokenSizes();
            expect(signSpy).not.toHaveBeenCalled();
            expect(instance.getTokenHeaderSize()).to.be.greaterThan(0);
            expect(instance.getTokenSignatureSize()).to.be.greaterThan(0);
            expect(instance.getMaxPayloadSize()).to.be.greaterThan(0);
            signSpy.mockRestore();
        });
    });

    describe("calculateBase64Length", () => {
        it("calculates base64 length of string payload", () => {
            const payload = "test";
            const length = JsonWebToken.calculateBase64Length(payload);
            expect(length).to.be.greaterThan(0);
        });

        it("calculates base64 length of object payload", () => {
            const payload = { foo: "bar", baz: 123 };
            const length = JsonWebToken.calculateBase64Length(payload);
            expect(length).to.be.greaterThan(0);
        });
    });

    describe("isTokenExpired", () => {
        // Fixed current time for consistent testing (e.g., Jan 1, 2021 00:00:00 GMT)
        const fixedTime = 1609459200 * SECONDS_1_MS; // in milliseconds

        beforeEach(() => {
            jest.spyOn(Date, "now").mockReturnValue(fixedTime);
        });

        afterEach(() => {
            jest.restoreAllMocks();
        });

        it("returns true when exp is undefined", () => {
            const payload: JwtPayload = { foo: "bar" };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns true when exp is NaN", () => {
            const payload: JwtPayload = { exp: NaN };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns true when exp is Infinity", () => {
            const payload: JwtPayload = { exp: Infinity };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns true when exp is -Infinity", () => {
            const payload: JwtPayload = { exp: -Infinity };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns true when exp is less than current time", () => {
            const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) - 1 };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns false when exp is exactly equal to current time", () => {
            const payload: JwtPayload = { exp: fixedTime / SECONDS_1_MS };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(false);
        });

        it("returns false when exp is greater than current time", () => {
            const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) + 1 };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(false);
        });

        it("handles payload with additional properties correctly", () => {
            const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) + 1, foo: "bar", baz: 123 };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(false);
        });

        it("handles payload with exp as zero (expired)", () => {
            const payload: JwtPayload = { exp: 0 };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("handles payload with negative exp (expired)", () => {
            const payload: JwtPayload = { exp: -100 };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns true when payload is null", () => {
            const payload = null;
            // @ts-ignore Testing runtime scenario
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("returns true when payload is undefined", () => {
            const payload = undefined;
            // @ts-ignore Testing runtime scenario
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(true);
        });

        it("handles non-integer exp values correctly", () => {
            const payload: JwtPayload = { exp: (fixedTime / SECONDS_1_MS) + 123.456 };
            expect(JsonWebToken.isTokenExpired(payload)).to.equal(false);
        });
    });

    describe("getMaxAge", () => {
        it("calculates max age from payload", () => {
            const currentTime = Date.now();
            const exp = Math.floor((currentTime + 60_000) / SECONDS_1_MS);
            const payload = { exp };
            const maxAge = JsonWebToken.getMaxAge(payload as any);
            expect(Math.abs(maxAge - 60_000)).to.be.at.most(SECONDS_1_MS);
        });

        it("returns negative max age if token already expired", () => {
            const currentTime = Date.now();
            const exp = Math.floor((currentTime - 60000) / SECONDS_1_MS);
            const payload = { exp };
            const maxAge = JsonWebToken.getMaxAge(payload as any);
            expect(maxAge).to.be.lessThan(0);
        });
    });

    describe("sign", () => {
        let instance: JsonWebToken;
        let privateKey: string;
        beforeEach(() => {
            instance = JsonWebToken.get();
            const { privateKey: privKey } = generateKeyPairSync("rsa", {
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
            instance.setupTestEnvironment({ privateKey, publicKey: "" });
        });

        it("signs token with default expiration", () => {
            const payload = { foo: "bar" };
            const token = instance.sign(payload);
            const decoded = jwt.verify(token, privateKey, { algorithms: ["RS256"] });
            expect(decoded).toMatchObject(payload);
        });

        it("signs token and keeps existing expiration", () => {
            const payload = { foo: "bar", exp: JsonWebToken.createExpirationTime(10_000) };
            const token = instance.sign(payload, true);
            const decoded = jwt.verify(token, privateKey, { algorithms: ["RS256"] });
            expect(decoded).toMatchObject(payload);
        });

        it("signs token and overrides expiration if keepExpiration is false", () => {
            const payload = { foo: "bar", exp: JsonWebToken.createExpirationTime(10_000) };
            const token = instance.sign(payload, false);
            const decoded = jwt.verify(token, privateKey, { algorithms: ["RS256"] }) as JwtPayload;
            expect(decoded.foo).to.equal("bar");
            const expectedExp = JsonWebToken.createExpirationTime(REFRESH_TOKEN_EXPIRATION_MS);
            expect(Math.abs(decoded.exp! - expectedExp)).to.be.at.most(SECONDS_1_MS);
        });

        it("throws error if privateKey is missing", () => {
            instance.setupTestEnvironment({ privateKey: "", publicKey: "" });
            const payload = { foo: "bar" };
            expect(() => instance.sign(payload)).to.throw();
        });

        it("does not set exp when keepExpiration is true", () => {
            const jwtSignSpy = jest.spyOn(jwt, "sign");
            const payload = { foo: "bar" };
            instance.sign(payload, true);
            expect(jwtSignSpy).toHaveBeenCalledWith(
                {
                    ...payload,
                    // No "exp"
                },
                privateKey,
                {
                    algorithm: "RS256",
                },
            );
            jwtSignSpy.mockRestore();
        });

        it("sets exp when keepExpiration is false", () => {
            const jwtSignSpy = jest.spyOn(jwt, "sign");
            const payload = { foo: "bar" };
            instance.sign(payload, false);
            expect(jwtSignSpy).toHaveBeenCalledWith(
                {
                    ...payload,
                    exp: JsonWebToken.createExpirationTime(REFRESH_TOKEN_EXPIRATION_MS),
                },
                privateKey,
                {
                    algorithm: "RS256",
                },
            );
            jwtSignSpy.mockRestore();
        });

        it("throws error if jwt.sign throws an error", () => {
            const signSpy = jest.spyOn(jwt, "sign").mockImplementation(() => {
                throw new Error("jwt.sign failed");
            });
            expect(() => instance.sign({})).toThrow("jwt.sign failed");
            signSpy.mockRestore();
        });
    });

    describe("verify", () => {
        let privateKey: string;
        let publicKey: string;

        beforeEach(() => {
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
            JsonWebToken.get().setupTestEnvironment({ privateKey, publicKey });
        });

        it("should resolve with payload for a valid token", async () => {
            const payload = { userId: "12345", role: "admin" };
            const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1h" });

            await expect(JsonWebToken.get().verify(token)).resolves.toMatchObject(payload);
        });

        it("should reject if token is invalid", async () => {
            const invalidToken = "invalid.token.here";

            await expect(JsonWebToken.get().verify(invalidToken)).rejects.toThrow(jwt.JsonWebTokenError);
        });

        describe("expired", () => {
            describe("reject", () => {
                it("negative expiresIn", async () => {
                    const payload = { userId: "12345", role: "admin" };
                    const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "-10s" });

                    await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.TokenExpiredError);
                });
                it("negative exp", async () => {
                    const payload = { userId: "12345", role: "admin", exp: -1 };
                    const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                    await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.TokenExpiredError);
                });
                it("zero exp", async () => {
                    const payload = { userId: "12345", role: "admin", exp: 0 };
                    const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                    await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.TokenExpiredError);
                });
                it("exp less than current time", async () => {
                    const payload = { userId: "12345", role: "admin", exp: (Date.now() / SECONDS_1_MS) - 1 };
                    const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                    await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.TokenExpiredError);
                });
            });
            describe("resolve", () => {
                it("positive expiresIn", async () => {
                    const payload = { userId: "12345", role: "admin" };
                    const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1s" });

                    await expect(JsonWebToken.get().verify(token)).resolves.toMatchObject(payload);
                });
                it("exp greater than current time", async () => {
                    const payload = { userId: "12345", role: "admin", exp: (Date.now() / SECONDS_1_MS) + 1 };
                    const token = jwt.sign(payload, privateKey, { algorithm: "RS256" });

                    await expect(JsonWebToken.get().verify(token)).resolves.toMatchObject(payload);
                });
            });
        });

        it("should reject if public key is incorrect", async () => {
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

            JsonWebToken.get().setupTestEnvironment({ privateKey, publicKey: incorrectPublicKey });
            await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.JsonWebTokenError);
        });

        it("should reject if token signed with different algorithm", async () => {
            const hmacKey = "mysecret";
            const payload = { userId: "12345", role: "admin" };
            const token = jwt.sign(payload, hmacKey, { algorithm: "HS256", expiresIn: "1h" });

            await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.JsonWebTokenError);
        });

        it("should reject if token is missing", async () => {
            const token = "";

            await expect(JsonWebToken.get().verify(token)).rejects.toThrow(jwt.JsonWebTokenError);
        });

        it("should reject if token is undefined", async () => {
            const token = undefined as unknown as string;

            await expect(JsonWebToken.get().verify(token)).rejects.toThrow(Error);
        });

        it("should reject with error if jwt.verify returns undefined payload", async () => {
            const payload = { userId: "12345", role: "admin" };
            const token = jwt.sign(payload, privateKey, { algorithm: "RS256", expiresIn: "1h" });

            const verifySpy = jest.spyOn(jwt, "verify").mockImplementation((token, publicKey, options, callback) => {
                (callback as jwt.VerifyCallback)(null, undefined);
            });

            await expect(JsonWebToken.get().verify(token)).rejects.toThrow(expect.any(Error));

            verifySpy.mockRestore();
        });
    });

    describe("addToCookies", () => {
        let res: Response;
        beforeEach(() => {
            res = {
                headersSent: false,
                cookie: jest.fn(),
            } as unknown as Response;
        });

        it("adds token to cookies when headers not sent", () => {
            const token = "testToken";
            const maxAge = 1000;
            JsonWebToken.addToCookies(res, token, maxAge);
            expect(res.cookie).toHaveBeenCalledWith(
                COOKIE.Jwt,
                token,
                {
                    ...JsonWebToken.getJwtCookieOptions(),
                    maxAge,
                },
            );
        });

        it("does not add token to cookies and logs error when headers have been sent", () => {
            res.headersSent = true;
            const token = "testToken";
            const maxAge = 1000;
            JsonWebToken.addToCookies(res, token, maxAge);
            expect(res.cookie).not.toHaveBeenCalled();
        });

        it("throws error if res.cookie throws an error", () => {
            res.cookie = jest.fn(() => { throw new Error("cookie failed"); });
            expect(() => {
                JsonWebToken.addToCookies(res, "token", 1000);
            }).toThrow("cookie failed");
        });
    });
});
