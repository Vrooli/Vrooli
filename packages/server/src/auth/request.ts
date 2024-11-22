import { DEFAULT_LANGUAGE } from "@local/shared";
import { type Request } from "express";
import pkg from "lodash";
import { CustomError } from "../events/error";
import { SessionData, SessionUserToken } from "../types";
import { SessionService } from "./session";

const { escapeRegExp } = pkg;

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

type AssertRequestFromResult<T extends RequestConditions> = T extends { isUser: true } | { isOfficialUser: true } ? SessionUserToken : undefined;

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

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): RequestService {
        if (!RequestService.instance) {
            RequestService.instance = new RequestService();
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
    };

    /**
     * Validate if the given string is a valid domain name 
     * (e.g. mysite.com, www.mysite.com, subdomain.mysite.com).
     * @param domain The domain name to validate.
     * @returns True if valid, false otherwise.
     */
    static isValidDomain(domain: string): boolean {
        if (domain.length > 253) return false;
        return RequestService.domainRegex.test(domain);
    };

    /**
     * Origins which are allowed to make requests without an API key.
     * @returns An array of strings and regular expressions
     */
    safeOrigins(): Array<string | RegExp> {
        if (this.cachedOrigins !== null) {
            return this.cachedOrigins;
        }

        const origins: Array<string | RegExp> = [];
        const siteIp = process.env.SITE_IP;
        if (process.env.VITE_SERVER_LOCATION === "local") {
            origins.push(RequestService.localhostRegex, RequestService.localhostIpRegex, "https://studio.apollographql.com");
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
    };

    /**
     * Checks if a request comes from a safe origin.
     * @param req The request to check
     * @returns True if the request comes from a safe origin
     */
    isSafeOrigin(req: Request): boolean {
        // Allow all on development. This ensures that graphql-generate and other 
        // dev tools work properly.
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
    };

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
        const userAgent = req.headers['user-agent'] || 'Unknown';

        // Optionally, include other headers or information as needed
        // For example, you can include the Accept-Language header
        const acceptLanguage = req.headers['accept-language'] || 'Unknown';

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
    static assertRequestFrom<Conditions extends RequestConditions>(req: { session: SessionData }, conditions: Conditions): AssertRequestFromResult<Conditions> {
        const { session } = req;
        // Determine if user data is found in the request
        const userData = SessionService.getUser(session);
        const hasUserData = session.isLoggedIn === true && Boolean(userData);
        // Determine if api token is supplied
        const hasApiToken = session.apiToken === true;
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
}
