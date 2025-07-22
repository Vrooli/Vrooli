import { AccountStatus, ApiKeyPermission, DAYS_1_MS, generatePK, generatePublicId, nanoid, SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import { type Request, type Response } from "express";
import { AuthService, AuthTokensService } from "../auth/auth.js";
import { type UserDataForPasswordAuth } from "../auth/email.js";
import { JsonWebToken } from "../auth/jwt.js";
import { SessionService } from "../auth/session.js";
import { DbProvider } from "../db/provider.js";

export function loggedInUserNoPremiumData(): UserDataForPasswordAuth {
    return {
        id: generatePK(),
        handle: "test-user",
        lastLoginAttempt: new Date(),
        logInAttempts: 0,
        name: "Test User",
        profileImage: null,
        publicId: generatePublicId(),
        theme: "dark",
        status: AccountStatus.Unlocked,
        updatedAt: new Date(),
        auths: [{
            id: generatePK(),
            provider: "Password",
            hashed_password: "dummy-hash",
        }],
        emails: [{
            emailAddress: "test-user@example.com",
        }],
        phones: [],
        plan: null,
        creditAccount: null,
        languages: ["en"],
        sessions: [],
    };
}

/**
 * Creates a mock session object that simulates a logged-in user
 * @param userData The user to create a mock session for. Must be password authenticated
 * @param req The request object to use for the session
 * @param res The response object to use for the session
 * @returns A mock session object with appropriate session data
 */
async function createMockSession(userData: UserDataForPasswordAuth, req: Request, res: Response) {
    const authId = userData.auths?.find(a => a.provider === "Password")?.id;
    if (!authId) {
        throw new Error("User must be password authenticated to create a mock session");
    }

    // Create a session with timestamps that ensure it won't expire during tests
    const now = new Date();
    const future = new Date(now.getTime() + DAYS_1_MS);

    const sessionData: UserDataForPasswordAuth["sessions"][0] = {
        id: generatePK(),
        device_info: "test-device-info",
        ip_address: "127.0.0.1",
        last_refresh_at: now,
        revokedAt: null,
        auth: { id: authId, provider: "Password" },
    };
    await DbProvider.get().user_auth.upsert({
        where: { id: authId },
        create: { id: authId, provider: "Password", hashed_password: "dummy-hash", user: { connect: { id: userData.id } } },
        update: { hashed_password: "dummy-hash" },
    });
    await DbProvider.get().session.upsert({
        where: { id: sessionData.id },
        create: {
            id: sessionData.id,
            device_info: sessionData.device_info,
            ip_address: sessionData.ip_address,
            expires_at: future,
            last_refresh_at: now,
            user: { connect: { id: BigInt(userData.id) } },
            auth: { connect: { id: authId } },
        },
        update: {
            expires_at: future,
            last_refresh_at: now,
        },
    });

    // Create session with proper structure
    const sessionUser = await SessionService.createUserSession(userData, sessionData);
    const session = {
        __typename: "Session",
        isLoggedIn: true,
        users: [sessionUser],
        // Add the access expiration so token won't be considered expired
        ...JsonWebToken.createAccessExpiresAt(),
    };

    // Generate token and set it in the response
    await AuthTokensService.generateSessionToken(res, session);

    return session;
}

/**
 * Creates a mock request object for testing
 * @returns A minimal Express Request object with required properties
 */
export function mockRequest(): Request {
    const result = {
        headers: {
            authorization: null,
            "accept-language": "en-US,en;q=0.9",
            origin: "http://localhost:3000",
        },
        cookies: {},
        ip: "127.0.0.1",
        session: undefined,
        query: {} as Record<string, any>,
        header(name: string) {
            const key = name.toLowerCase();
            const value = (this.headers as any)[key];
            if (Array.isArray(value)) {
                return value[0];
            }
            return value;
        },
    };

    return result as unknown as Request;
}

/**
 * Creates a mock response object that can be used for testing endpoints
 * that need to interact with the response (like setting cookies)
 * @returns A mock Express Response object with common methods
 */
export function mockResponse(): Response {
    const result = {
        // Response status and data tracking
        statusCode: 200,
        jsonData: null as any,
        headersSent: false,

        // Cookie tracking
        cookies: {} as Record<string, string>,

        // Common response methods
        status(code: number) {
            this.statusCode = code;
            return this;
        },
        json(data: any) {
            this.jsonData = data;
            return this;
        },
        send(_data: any) {
            return this;
        },
        cookie(name: string, value: string, _options?: any) {
            this.cookies[name] = value;
            return this;
        },
        clearCookie(name: string) {
            delete this.cookies[name];
            return this;
        },
    };

    // Type assertion to treat our mock as an Express Response
    return result as unknown as Response;
}

/**
 * Creates a request and response pair with proper authentication for testing
 * @param userData User data to authenticate with - can be full UserDataForPasswordAuth or simple object with id
 * @returns Object containing req and res objects ready for use in tests
 */
export async function mockAuthenticatedSession(userData: UserDataForPasswordAuth | { id: string; [key: string]: any }) {
    const req = mockRequest();
    const res = mockResponse();

    // If userData is a simple object with just id, create a full UserDataForPasswordAuth object
    let fullUserData: UserDataForPasswordAuth;
    if ("emails" in userData && "auths" in userData) {
        // It's already a full UserDataForPasswordAuth object
        fullUserData = userData as UserDataForPasswordAuth;
    } else {
        // It's a simple object, create a full UserDataForPasswordAuth object
        fullUserData = {
            ...loggedInUserNoPremiumData(),
            id: BigInt(userData.id),
            publicId: userData.publicId || generatePublicId(),
            name: userData.name || "Test User",
            handle: userData.handle || "test-user",
            // Override any other properties provided
            ...userData,
        };
    }

    // Set up the session with the provided user data
    await createMockSession(fullUserData, req, res);

    // Transfer cookies from response to request to simulate a real browser flow
    // where cookies set in a previous response are sent in subsequent requests
    req.cookies = (res as any).cookies;

    // Run the authentication middleware to properly set up the request
    await new Promise<void>((resolve) => {
        AuthService.authenticateRequest(req, res, () => {
            resolve();
        });
    });

    // If users array was provided, set it in the session
    if ("users" in userData && Array.isArray(userData.users)) {
        if (req.session) {
            req.session.users = userData.users as any;
        }
    }

    return { req, res };
}

/**
 * Creates a mock session object that simulates an API key
 * @param apiToken The API key to use for the session
 * @param permissions The permissions of the API key
 * @param userData User data to authenticate with
 * @returns A mock session object with appropriate session data
 */
export async function mockApiSession(apiToken: string, permissions: Record<ApiKeyPermission, boolean>, userData: UserDataForPasswordAuth) {
    const req = mockRequest();
    const res = mockResponse();

    // Generate API token - ensure ID is a string
    const userId = typeof userData.id === "bigint" ? userData.id.toString() : userData.id.toString();
    await AuthTokensService.generateApiToken(res, apiToken, permissions, userId);

    // Transfer cookies from response to request to simulate a real browser flow
    req.cookies = (res as any).cookies;

    // Run the authentication middleware to properly set up the request
    await new Promise<void>((resolve) => {
        AuthService.authenticateRequest(req, res, () => {
            resolve();
        });
    });

    return { req, res, apiToken };
}

export async function mockLoggedOutSession() {
    const req = mockRequest();
    const res = mockResponse();

    req.session = {
        fromSafeOrigin: true,
        languages: ["en"],
    };

    return { req, res };
}

export function mockReadPublicPermissions(): Record<ApiKeyPermission, boolean> {
    return {
        [ApiKeyPermission.ReadPublic]: true,
        [ApiKeyPermission.ReadPrivate]: false,
        [ApiKeyPermission.ReadAuth]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.WritePrivate]: false,
    };
}

export function mockReadPrivatePermissions(): Record<ApiKeyPermission, boolean> {
    return {
        [ApiKeyPermission.ReadPublic]: true, // Typically also true when ReadPrivate is true
        [ApiKeyPermission.ReadPrivate]: true,
        [ApiKeyPermission.ReadAuth]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.WritePrivate]: false,
    };
}

export function mockWritePrivatePermissions(): Record<ApiKeyPermission, boolean> {
    return {
        [ApiKeyPermission.ReadPublic]: true, // Typically also true when WritePrivate is true
        [ApiKeyPermission.ReadPrivate]: true, // Typically also true when WritePrivate is true
        [ApiKeyPermission.ReadAuth]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.WritePrivate]: true,
    };
}

export function mockReadAuthPermissions(): Record<ApiKeyPermission, boolean> {
    return {
        [ApiKeyPermission.ReadPublic]: true, // Typically also true when ReadAuth is true
        [ApiKeyPermission.ReadPrivate]: true, // Typically also true when ReadAuth is true
        [ApiKeyPermission.ReadAuth]: true,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.WritePrivate]: false,
    };
}

export function mockWriteAuthPermissions(): Record<ApiKeyPermission, boolean> {
    // Typically all permissions are true when WriteAuth is true
    return {
        [ApiKeyPermission.ReadPublic]: true,
        [ApiKeyPermission.ReadPrivate]: true,
        [ApiKeyPermission.ReadAuth]: true,
        [ApiKeyPermission.WriteAuth]: true,
        [ApiKeyPermission.WritePrivate]: true,
    };
}

export function defaultPublicUserData() {
    return {
        id: generatePK(),
        publicId: generatePublicId(),
        name: "Test User",
        handle: nanoid(),
        status: "Unlocked" as const,
        isBot: false,
        isBotDepictingPerson: false,
        isPrivate: false,
        auths: { create: [{ id: generatePK(), provider: "Password", hashed_password: "dummy-hash" }] },
    };
}

/**
 * Seeds a mock admin user
 * @returns The admin user
 */
export async function seedMockAdminUser() {
    const admin = await DbProvider.get().user.upsert({
        where: { publicId: SEEDED_PUBLIC_IDS.Admin },
        update: {},
        create: {
            ...defaultPublicUserData(),
            id: generatePK(),
            publicId: SEEDED_PUBLIC_IDS.Admin,
            name: "Admin User",
            handle: "admin",
        },
    });
    return admin;
}
