import { DAYS_1_S, DEFAULT_LANGUAGE, SessionUser } from "@local/shared";
import { type Request } from "express";
import fs from "fs";
import pkg from "lodash";
import { RedisClientType } from "redis";
import { Socket } from "socket.io";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { initializeRedis } from "../redisConn.js";
import { SessionData } from "../types.js";
import { SessionService } from "./session.js";

const { escapeRegExp } = pkg;

const DEFAULT_RATE_LIMIT = 250;
const DEFAULT_RATE_LIMIT_WINDOW_S = DAYS_1_S;
const MAX_DOMAIN_LENGTH = 253;

const tokenBucketScriptFile = `${process.env.PROJECT_DIR}/packages/server/${process.env.NODE_ENV === "development" ? "src" : "dist"}/utils/tokenBucketScript.lua`;

export type RequestConditions = {
    /**
     * Checks if the request is coming from an API token directly
     */
    isApiRoot?: boolean;
    /**
     * Checks if the request is coming from a user logged in via an API token, or the official Vrooli app/website
     * This allows other services to use Vrooli as a backend, in a way that 
     * we can price it accordingly.
     */
    isUser?: boolean;
    /**
     * Checks if the request is coming from a user logged in via the official Vrooli app/website
     */
    isOfficialUser?: boolean;
}

type AssertRequestFromResult<T extends RequestConditions> = T extends { isUser: true } | { isOfficialUser: true } ? SessionUser : undefined;

export interface RateLimitProps {
    /**
     * Maximum number of requests allowed per window, tied to API key (if not made from a safe origin)
     */
    maxApi?: number;
    /**
     * Maximum number of requests allowed per window, tied to IP address (if API key is not supplied)
     */
    maxIp?: number;
    /**
     * Maximum number of requests allowed per window, tied to user (regardless of origin)
     */
    maxUser?: number;
    req: Request;
    window?: number;
}

interface SocketRateLimitProps {
    maxIp?: number;
    maxUser?: number;
    window?: number;
    socket: Socket;
}

export class RequestService {
    private static instance: RequestService;

    private cachedOrigins: Array<string | RegExp> | null = null;

    private static ipv4Regex = /^((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])$/;
    private static ipv6Pattern = `
    (
        ([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|          # 1:2:3:4:5:6:7:8
        ([0-9a-fA-F]{1,4}:){1,7}:|                         # 1::                              1:2:3:4:5:6:7::
        ([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|         # 1::8             1:2:3:4:5:6::8  1:2:3:4:5:6::8
        ([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|  # 1::7:8           1:2:3:4:5::7:8  1:2:3:4:5::8
        ([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|  # 1::6:7:8         1:2:3:4::6:7:8  1:2:3:4::8
        ([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|  # 1::5:6:7:8       1:2:3::5:6:7:8  1:2:3::8
        ([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|  # 1::4:5:6:7:8     1:2::4:5:6:7:8  1:2::8
        [0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|       # 1::3:4:5:6:7:8   1::3:4:5:6:7:8  1::8  
        :((:[0-9a-fA-F]{1,4}){1,7}|:)|                     # ::2:3:4:5:6:7:8  ::2:3:4:5:6:7:8 ::8       ::     
        fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|     # fe80::7:8%eth0   fe80::7:8%1     (link-local IPv6 addresses with zone index)
        ::(ffff(:0{1,4}){0,1}:){0,1}
        ((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}
        (25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|          # ::255.255.255.255   ::ffff:255.255.255.255  ::ffff:0:255.255.255.255  (IPv4-mapped IPv6 addresses and IPv4-translated addresses)
        ([0-9a-fA-F]{1,4}:){1,4}:
        ((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}
        (25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])           # 2001:db8:3:4::192.0.2.33  64:ff9b::192.0.2.33 (IPv4-Embedded IPv6 Address)
    )
    `.replace(/\s*#.*$/gm, "").replace(/\s+/g, "");
    private static ipv6Regex = new RegExp(RequestService.ipv6Pattern);
    private static domainRegex = /^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,63}$/i;
    private static localhostRegex = /^http:\/\/localhost(?::[0-9]+)?$/;
    private static localhostIpRegex = /^http:\/\/192\.168\.[0-9]{1,3}\.[0-9]{1,3}(?::[0-9]+)?$/;


    static tokenBucketScript = "";
    private static tokenBucketScriptSha: string | null = null;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): RequestService {
        if (!RequestService.instance) {
            RequestService.instance = new RequestService();
            // Load the token bucket script
            try {
                RequestService.tokenBucketScript = fs.readFileSync(tokenBucketScriptFile, "utf8");
            } catch (error) {
                logger.error(`Could not find or read token bucket script file at ${tokenBucketScriptFile}`, { trace: "0369" });
            }
        }
        return RequestService.instance;
    }

    /**
     * Validate if the given string is a valid IP address.
     * @param ip The IP address to validate.
     * @returns True if valid, false otherwise.
     */
    static isValidIP(ip: string): boolean {
        return RequestService.ipv4Regex.test(ip) || RequestService.ipv6Regex.test(ip);
    }

    /**
     * Validate if the given string is a valid domain name 
     * (e.g. mysite.com, www.mysite.com, subdomain.mysite.com).
     * @param domain The domain name to validate.
     * @returns True if valid, false otherwise.
     */
    static isValidDomain(domain: string): boolean {
        if (domain.length > MAX_DOMAIN_LENGTH) return false;
        return RequestService.domainRegex.test(domain);
    }

    /**
     * Origins which are allowed to make requests without an API key.
     * @returns An array of strings and regular expressions
     */
    safeOrigins(): Array<string | RegExp> {
        if (Array.isArray(this.cachedOrigins) && this.cachedOrigins.length > 0) {
            return this.cachedOrigins;
        }

        const origins: Array<string | RegExp> = [];
        const siteIp = process.env.SITE_IP;
        if (process.env.VITE_SERVER_LOCATION === "local") {
            origins.push(RequestService.localhostRegex, RequestService.localhostIpRegex);
        }
        const domains = (process.env.VIRTUAL_HOST ?? "").split(",");
        for (const domain of domains) {
            if (!RequestService.isValidDomain(domain)) continue;
            origins.push(new RegExp(`^http(s)?://${escapeRegExp(domain)}$`));
        }
        if (siteIp && RequestService.isValidIP(siteIp)) {
            origins.push(new RegExp(`^http(s)?://${escapeRegExp(siteIp)}(?::[0-9]+)?$`));
        }

        this.cachedOrigins = origins;
        return origins;
    }

    /**
     * Checks if a request comes from a safe origin.
     * @param req The request to check
     * @returns True if the request comes from a safe origin
     */
    isSafeOrigin(req: Request): boolean {
        // Allow all on development. This ensures that dev tools work properly.
        if (process.env.NODE_ENV === "development") return true;
        const origins = this.safeOrigins();
        let origin = req.headers.origin;
        // Sometimes the origin is undefined. Luckily, we can parse it from the referer.
        if (!origin) {
            if (req.headers.referer) {
                const refererUrl = new URL(req.headers.referer);
                origin = refererUrl.origin;
            }
        }
        if (origin === undefined) {
            return false;
        }
        for (const o of origins) {
            if (o instanceof RegExp) {
                if (o.test(origin)) {
                    return true;
                }
            }
            else if (o === origin) {
                return true;
            }
        }
        return false;
    }

    /**
     * WARNING: For testing purposes only
     */
    resetCachedOrigins(): void {
        this.cachedOrigins = null;
    }

    /**
     * Extracts basic device information from the incoming HTTP request. 
     * Useful for detecting when a user logs in from a new device
     *
     * @param req - The incoming HTTP request object.
     * @returns A string containing the device's User-Agent and Accept-Language information.
     */
    static getDeviceInfo(req: Request): string {
        // Retrieve the User-Agent header from the request
        const userAgent = req.headers["user-agent"] || "Unknown";

        // Optionally, include other headers or information as needed
        // For example, you can include the Accept-Language header
        const acceptLanguage = req.headers["accept-language"] || "Unknown";

        // Combine the information into a single string
        const deviceInfo = `User-Agent: ${userAgent}; Accept-Language: ${acceptLanguage}`;

        return deviceInfo;
    }

    /**
     * Parses a request's accept-language header
     * @param req The request
     * @returns A list of languages without any subtags
     */
    static parseAcceptLanguage(req: { headers: Record<string, any> }): string[] {
        const acceptString = req.headers["accept-language"];
        // Default to english if not found or a wildcard
        if (!acceptString || typeof acceptString !== "string" || acceptString === "*") return [DEFAULT_LANGUAGE];
        // Strip q values
        let acceptValues = acceptString.split(",").map((lang: string) => lang.split(";")[0]);
        // Remove subtags
        acceptValues = acceptValues.map((lang: string) => lang.split("-")[0]);
        return acceptValues;
    }

    /**
     * Asserts that a request meets the specifiec requirements TODO need better api token validation, like uuidValidate
     * @param req Object with request data
     * @param conditions The conditions to check
     * @returns user data, if isUser or isOfficialUser is true
     * @throws CustomError if conditions are not met
     */
    static assertRequestFrom<Conditions extends RequestConditions>(
        req: { session: Pick<SessionData, "apiToken" | "fromSafeOrigin" | "isLoggedIn" | "languages"> & { users?: Pick<SessionUser, "id" | "languages">[] | null | undefined } },
        conditions: Conditions,
    ): AssertRequestFromResult<Conditions> {
        const { session } = req;
        // Determine if user data is found in the request
        const userData = SessionService.getUser(session);
        const hasUserData = session.isLoggedIn === true && Boolean(userData);
        // Determine if api token is supplied
        const hasApiToken = typeof session.apiToken === "string" && session.apiToken.length > 0;
        // Check isApiRoot condition
        if (conditions.isApiRoot !== undefined) {
            const isApiRoot = hasApiToken && !hasUserData;
            if (conditions.isApiRoot === true && !isApiRoot) throw new CustomError("0265", "MustUseApiToken");
            if (conditions.isApiRoot === false && isApiRoot) throw new CustomError("0266", "MustNotUseApiToken");
        }
        // Check isUser condition
        if (conditions.isUser !== undefined) {
            const isUser = hasUserData && (hasApiToken || session.fromSafeOrigin === true);
            if (conditions.isUser === true && !isUser) throw new CustomError("0267", "NotLoggedIn");
            if (conditions.isUser === false && isUser) throw new CustomError("0268", "NotLoggedIn");
        }
        // Check isOfficialUser condition
        if (conditions.isOfficialUser !== undefined) {
            const isOfficialUser = hasUserData && !hasApiToken && session.fromSafeOrigin === true;
            if (conditions.isOfficialUser === true && !isOfficialUser) throw new CustomError("0269", "NotLoggedInOfficial");
            if (conditions.isOfficialUser === false && isOfficialUser) throw new CustomError("0270", "NotLoggedInOfficial");
        }
        return conditions.isUser === true || conditions.isOfficialUser === true ? userData as any : undefined;
    }

    /**
     * Builds a key for rate limiting a normal (non-socket) request
     * @param req The request to build the key from
     * @returns The key base
     */
    private static buildKeyBase(req: Request): string {
        let keyBase = "rate-limit:";

        // For GraphQL requests, use the operation name
        if (req.body?.operationName) {
            keyBase += `${req.body.operationName}:`;
        }
        // For REST requests, use the route path and method
        else if (req.route) {
            keyBase += `${req.route.path}:${req.method}:`;
        }
        // For other requests (typically when req is mocked by a task queue), use the path
        else {
            keyBase += `${req.path}:`;
        }

        return keyBase;
    }

    /**
     * Builds a key for rate limiting an IP address
     * @param req The request to build the key from
     * @returns The key base
     */
    private static buildIpKey(req: Request): string {
        return `${RequestService.buildKeyBase(req)}ip:${req.ip}`;
    }

    /**
     * Builds a key for rate limiting an API token
     * @param req The request to build the key from
     * @returns The key base
     */
    private static buildApiKey(req: Request): string {
        return `${RequestService.buildKeyBase(req)}api:${req.session.apiToken}`;
    }

    /**
     * Builds a key for rate limiting a user
     * @param req The request to build the key from
     * @param userData The user data to build the key from
     * @returns The key base
     */
    private static buildUserKey(req: Request, userData: SessionUser): string {
        return `${RequestService.buildKeyBase(req)}user:${userData.id}`;
    }

    /**
     * Builds a key base for rate limiting a socket request
     * @param socket The socket to build the key from
     * @returns The key base
     */
    private static buildSocketKeyBase(socket: Socket): string {
        return `rate-limit:${socket.id}:`;
    }

    /**
     * Builds a key for rate limiting an IP address for a socket request
     * @param socket The socket to build the key from
     * @returns The key base
     */
    private static buildSocketIpKey(socket: Socket): string {
        return `${RequestService.buildSocketKeyBase(socket)}ip:${socket.req.ip}`;
    }

    /**
     * Builds a key for rate limiting a user for a socket request
     * @param socket The socket to build the key from
     * @param userData The user data to build the key from
     * @returns The key base
     */
    private static buildSocketUserKey(socket: Socket, userData: SessionUser): string {
        return `${RequestService.buildSocketKeyBase(socket)}user:${userData.id}`;
    }

    /**
     * Applies a rate limit check to keys in redis. 
     * Throws an error if the limit is exceeded.
     * @param client The redis client
     * @param keys The keys to check
     * @param maxTokensList The maximum number of tokens allowed in the bucket 
     * for each key, in order
     * @param refillRates The rate at which the bucket refills for each key, in order
     */
    async checkRateLimit(
        client: RedisClientType | null,
        keys: string[],
        maxTokensList: number[],
        refillRates: number[],
    ) {
        if (!client) {
            return;
        }

        const nowMs = Date.now();

        const args: string[] = [];

        // Build ARGV
        for (let i = 0; i < keys.length; i++) {
            args.push(maxTokensList[i].toString());
            args.push(refillRates[i].toString());
        }

        // Append nowMs
        args.push(nowMs.toString());

        // Result is an array of [allowed1, waitTimeMs1, allowed2, waitTimeMs2, ...]  
        let result: number[] | undefined;
        if (RequestService.tokenBucketScriptSha) {
            try {
                result = await client.evalSha(RequestService.tokenBucketScriptSha, {
                    keys,
                    arguments: args,
                }) as number[];
            } catch (error) {
                if (!(error instanceof Error && error.message.startsWith("NOSCRIPT"))) {
                    throw error;
                }
                // If NOSCRIPT, proceed to load the script
            }
        }
        if (result === undefined) {
            RequestService.tokenBucketScriptSha = await client.scriptLoad(RequestService.tokenBucketScript);
            result = await client.evalSha(RequestService.tokenBucketScriptSha, {
                keys,
                arguments: args,
            }) as number[];
        }

        // Check if any of the rate limits are exceeded
        for (let i = 0; i < keys.length; i++) {
            const allowed = result[2 * i];
            const waitTimeMs = result[2 * i + 1];
            if (allowed === 0) {
                throw new CustomError("0017", "RateLimitExceeded", { retryAfterMs: waitTimeMs });
            }
        }
    }

    /**
     * Middelware to rate limit the requests. 
     * Limits requests based on API token, account, and IP address.
     * Throws error if rate limit is exceeded.
     */
    async rateLimit({
        maxApi,
        maxIp,
        maxUser = DEFAULT_RATE_LIMIT,
        req,
        window = DEFAULT_RATE_LIMIT_WINDOW_S,
    }: RateLimitProps): Promise<void> {
        // If maxApi not supplied, use maxUser * 1000
        maxApi = maxApi ?? (maxUser * 1000);
        // If maxIp not supplied, use maxUser
        maxIp = maxIp ?? maxUser;

        // Parse request
        const hasApiToken = typeof req.session.apiToken === "string" && req.session.apiToken.length > 0;
        const userData = SessionService.getUser(req.session);
        const hasUserData = req.session.isLoggedIn === true && userData !== null;

        // Try connecting to redis
        try {
            const client = await initializeRedis();

            // Calculate refill rates
            const apiRefillRate = maxApi / window;
            const ipRefillRate = maxIp / window;
            const userRefillRate = maxUser / window;

            // Build arrays
            const keys: string[] = [];
            const maxTokensList: number[] = [];
            const refillRates: number[] = [];

            // Apply rate limit to API
            if (hasApiToken) {
                const apiKey = RequestService.buildApiKey(req);
                keys.push(apiKey);
                maxTokensList.push(maxApi);
                refillRates.push(apiRefillRate);
            }
            // Make sure that all non-API requests are from a safe origin
            else if (req.session.fromSafeOrigin === false) {
                throw new CustomError("0271", "MustUseApiToken", { keyBase: RequestService.buildKeyBase(req) });
            }

            // Apply rate limit to IP address
            const ipKey = RequestService.buildIpKey(req);
            keys.push(ipKey);
            maxTokensList.push(maxIp);
            refillRates.push(ipRefillRate);

            // Apply rate limit to user
            if (hasUserData) {
                const userKey = RequestService.buildUserKey(req, userData);
                keys.push(userKey);
                maxTokensList.push(maxUser);
                refillRates.push(userRefillRate);
            }

            // Call checkRateLimit with arrays
            await this.checkRateLimit(client, keys, maxTokensList, refillRates);
        }
        // If Redis fails, let the user through. It's not their fault. 
        catch (error) {
            logger.error("Error occured while connecting or accessing redis server", { trace: "0168", error });
            throw error;
        }
    }

    /**
     * Rate limit a socket handler. Similar to how you would rate limit an API endpoint. 
     * Returns instead of throws an error if rate limit is exceeded.
     */
    async rateLimitSocket({
        maxIp,
        maxUser = DEFAULT_RATE_LIMIT,
        window = DEFAULT_RATE_LIMIT_WINDOW_S,
        socket,
    }: SocketRateLimitProps): Promise<string | undefined> {
        // Retrieve user data from the socket
        const userData = SessionService.getUser(socket.session);
        const hasUserData = socket.session.isLoggedIn === true && userData !== null;
        // If maxIp not supplied, use maxUser
        maxIp = maxIp ?? maxUser;
        // Try connecting to redis
        try {
            const client = await initializeRedis();

            // Calculate refill rates
            const ipRefillRate = maxIp / window;
            const userRefillRate = maxUser / window;

            // Build arrays
            const keys: string[] = [];
            const maxTokensList: number[] = [];
            const refillRates: number[] = [];

            // Apply rate limit to IP address
            const ipKey = RequestService.buildSocketIpKey(socket);
            keys.push(ipKey);
            maxTokensList.push(maxIp);
            refillRates.push(ipRefillRate);

            // Apply rate limit to user
            if (hasUserData) {
                const userKey = RequestService.buildSocketUserKey(socket, userData);
                keys.push(userKey);
                maxTokensList.push(maxUser);
                refillRates.push(userRefillRate);
            }

            // Call checkRateLimit with arrays
            await this.checkRateLimit(client, keys, maxTokensList, refillRates);
        } catch (error) {
            console.error("Error occurred while connecting or accessing redis server", { trace: "0492", error });
            return (error as Error)?.message ?? "Rate limit exceeded";
        }
    }
}
