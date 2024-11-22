import { COOKIE, MINUTES_15_MS, SECONDS_1_MS, YEARS_1_MS } from "@local/shared";
import { type Response } from "express";
import jwt, { type JwtPayload } from "jsonwebtoken";
import { logger } from "../events/logger";
import { UI_URL_REMOTE } from "../server";
import { BasicToken } from "../types";

/**
 * How long to treat a session as valid before checking 
 * the database for revocation or renewal.
 * 
 * Should be short, but not too short to avoid unnecessary
 * database queries.
 */
export const ACCESS_TOKEN_EXPIRATION_MS = MINUTES_15_MS;
/**
 * When to consider a session expired and require re-authentication.
 * 
 * Should be very long, since users can revoke sessions at any time.
 */
export const REFRESH_TOKEN_EXPIRATION_MS = YEARS_1_MS;

interface JwtKeys {
    privateKey: string;
    publicKey: string;
}

type VerifiedJwt = JwtPayload & {
    exp: number;
};

export class JsonWebToken {
    private static instance: JsonWebToken;

    /** Maximum size of a JWT token. Limited by the JWT standard. */
    private static TOKEN_LENGTH_LIMIT = 4096;
    /** Buffer for additional data in the token that we didn't account for, and just to be safe */
    private static TOKEN_BUFFER = 200;

    // Size of the header and signature of a JWT token. Calculated once and stored for future use.
    private tokenHeaderSize = 0;
    private tokenSignatureSize = 0;
    // Maximum size of the payload of a JWT token. Calculated once and stored for future use.
    private maxPayloadSize = 0;

    // Keys used for signing and verifying JWT tokens
    private keys: JwtKeys = { privateKey: "", publicKey: "" };

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): JsonWebToken {
        if (!JsonWebToken.instance) {
            JsonWebToken.instance = new JsonWebToken();
            if (process.env.NODE_ENV !== "test") {
                // Load the keys for signing and verifying JWT tokens
                JsonWebToken.instance.loadKeys();
                // Calculate token sizes
                JsonWebToken.instance.calculateTokenSizes();
            }
        }
        return JsonWebToken.instance;
    }

    /**
     * Loads the keys for signing and verifying JWT tokens
     */
    loadKeys() {
        // If keys are already defined, return early
        if (this.keys.privateKey && this.keys.publicKey) {
            return;
        }

        // Load the keys from process.env. They may be in a single line, so replace \n with actual newlines
        const privateKey = process.env.JWT_PRIV ?? "";
        const publicKey = process.env.JWT_PUB ?? "";

        // Check if the keys are available and log an error if not
        if (privateKey.length <= 0) {
            logger.error("JWT private key not found");
        }
        if (publicKey.length <= 0) {
            logger.error("JWT public key not found");
        }

        // Store the keys for future use
        this.keys = { privateKey, publicKey };
    }

    /**
     * WARNING: For testing purposes only
     */
    setupTestEnvironment(keys: JwtKeys) {
        // Store the keys
        this.keys = keys;
        // Calculate token sizes
        this.calculateTokenSizes();
    }

    static getJwtCookieOptions() {
        return {
            httpOnly: true, // Prevents client-side JavaScript from accessing the cookie
            sameSite: "lax", // Prevents CSRF attacks
            secure: process.env.NODE_ENV === "production", // Only send cookie over HTTPS in production
            maxAge: ACCESS_TOKEN_EXPIRATION_MS, // Set expiration time
        } as const;
    }

    /**
     * Generates the minimum data required for a JWT token
     * @param iss The issuer of the token. This should be the URL of the UI
     * @param exp The expiration time of the token in seconds
     */
    basicToken(
        iss = UI_URL_REMOTE,
        exp?: number
    ): BasicToken {
        return {
            iat: Date.now(),
            iss,
            exp: exp ?? Math.floor((Date.now() + ACCESS_TOKEN_EXPIRATION_MS) / SECONDS_1_MS),
        };
    }

    /**
     * Calculates token and signature size for JWT tokens
     */
    calculateTokenSizes() {
        const isMissingTokenSize = this.tokenHeaderSize <= 0 || this.tokenSignatureSize <= 0 || this.maxPayloadSize <= 0;
        if (!isMissingTokenSize) {
            return;
        }
        const emptyToken = this.sign({});
        const [header, , signature] = emptyToken.split(".");
        this.tokenHeaderSize = Buffer.byteLength(header, "utf8");
        this.tokenSignatureSize = Buffer.byteLength(signature, "utf8");
        // Total allowed size (both name and jwt value) - name - jwt header size - jwt payload size - jwt signature size - options size - buffer for additional data (e.g. "=" between name and value, ";" after value, path, expires)
        this.maxPayloadSize =
            JsonWebToken.TOKEN_LENGTH_LIMIT
            - COOKIE.Jwt.length
            - this.tokenHeaderSize
            - this.tokenSignatureSize
            - JSON.stringify(JsonWebToken.getJwtCookieOptions()).length
            - JsonWebToken.TOKEN_BUFFER;
    }

    /**
     * Calculates the length of a base64url encoded string (without padding, as that's added to the 
     * full JWT [i.e. with header and signature]) from a given JWT payload.
     * @param payload The payload to be encoded. If an object is provided, it will be stringified.
     * @returns The length of the base64url encoded string without padding.
     */
    static calculateBase64Length(payload: string | object) {
        // Calculate base64 string from payload
        const base64Payload = Buffer.from(JSON.stringify(payload)).toString("base64");
        // Remove padding
        const withoutPadding = base64Payload.replace(/=/g, "");
        // Return size of base64 string
        return withoutPadding.length;
    }

    /**
     * Checks if a token is expired
     * @param payload The token payload to check
     * @returns True if the token is expired
     */
    static isTokenExpired(payload: JwtPayload): payload is VerifiedJwt {
        const currentTimeInSeconds = Math.floor(Date.now() / SECONDS_1_MS);
        return payload?.exp === undefined
            || !isFinite(payload.exp)
            || payload.exp <= 0
            || payload.exp < currentTimeInSeconds;
    }

    getTokenHeaderSize() {
        return this.tokenHeaderSize;
    }

    getTokenSignatureSize() {
        return this.tokenSignatureSize;
    }

    getMaxPayloadSize() {
        return this.maxPayloadSize;
    }

    /**
     * Calculates the maximum age of a token (how long until it expires),
     * in milliseconds from the current time.
     * @param payload The token payload
     * @returns The maximum age of the token in milliseconds
     */
    static getMaxAge(payload: VerifiedJwt): number {
        const currentTime = Date.now();
        const expirationTime = payload.exp * SECONDS_1_MS; // Convert seconds to milliseconds
        return expirationTime - currentTime;
    }

    /**
     * Signs a JWT token
     * @param payload The payload to sign
     * @param keepExpiration If true, the existing expiration time will be kept
     * @returns 
     */
    sign(payload: object, keepExpiration = false): string {
        const options: jwt.SignOptions = {
            algorithm: "RS256",
        };
        // Add new expiration time if requested
        if (!keepExpiration) {
            payload["exp"] = Math.floor((Date.now() + ACCESS_TOKEN_EXPIRATION_MS) / SECONDS_1_MS);
        }
        return jwt.sign(payload, this.keys.privateKey, options);
    }

    async verify(token: string): Promise<VerifiedJwt> {
        return new Promise((resolve, reject) => {
            jwt.verify(token, this.keys.publicKey, { algorithms: ["RS256"] }, (error, payload) => {
                if (error) {
                    reject(error);
                } else if (!payload || typeof payload === "string" || !payload.exp) {
                    reject(new Error("Unexpected undefined or string payload from jwt.verify"));
                } else {
                    resolve(payload as VerifiedJwt);
                }
            });
        });
    }

    decode(token: string): VerifiedJwt | null {
        try {
            const payload = jwt.decode(token) as VerifiedJwt | null;
            if (payload && typeof payload !== 'string') {
                return payload;
            } else {
                return null;
            }
        } catch (error) {
            return null;
        }
    }

    /**
     * Adds a token to cookies
     * @param res The response object
     * @param token The signed token
     * @param maxAge The maximum age of the token in milliseconds
     */
    static addToCookies(res: Response, token: string, maxAge: number) {
        // Headers may have already been sent, so we need to check
        if (res.headersSent) {
            logger.error("❗️ Headers have already been sent", { trace: "0613" });
            return;
        }
        return res.cookie(COOKIE.Jwt, token, {
            ...JsonWebToken.getJwtCookieOptions(),
            maxAge,
        });
    }
}
