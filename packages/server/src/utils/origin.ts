import { Request } from "express";
import pkg from "lodash";

const { escapeRegExp } = pkg;

let cachedOrigins: Array<string | RegExp> | null = null;

const ipv4Regex = /^((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])$/;
const ipv6Pattern = `
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
const ipv6Regex = new RegExp(ipv6Pattern);
const domainRegex = /^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,63}$/i;
const localhostRegex = /^http:\/\/localhost(?::[0-9]+)?$/;
const localhostIpRegex = /^http:\/\/192\.168\.[0-9]{1,3}\.[0-9]{1,3}(?::[0-9]+)?$/;

/**
 * Validate if the given string is a valid IP address.
 * @param {string} ip - The IP address to validate.
 * @returns {boolean} - True if valid, false otherwise.
 */
export const isValidIP = (ip: string): boolean => {
    return ipv4Regex.test(ip) || ipv6Regex.test(ip);
};

/**
 * Validate if the given string is a valid domain name 
 * (e.g. mysite.com, www.mysite.com, subdomain.mysite.com).
 * @param {string} domain - The domain name to validate.
 * @returns {boolean} - True if valid, false otherwise.
 */
export const isValidDomain = (domain: string): boolean => {
    if (domain.length > 253) return false;
    return domainRegex.test(domain);
};

/**
 * Origins which are allowed to make requests without an API key.
 * @returns An array of strings and regular expressions
 */
export const safeOrigins = (): Array<string | RegExp> => {
    if (cachedOrigins !== null) {
        return cachedOrigins;
    }

    const origins: Array<string | RegExp> = [];
    const siteIp = process.env.SITE_IP;
    if (process.env.VITE_SERVER_LOCATION === "local") {
        origins.push(localhostRegex, localhostIpRegex, "https://studio.apollographql.com");
    }
    const domains = (process.env.VIRTUAL_HOST ?? "").split(",");
    for (const domain of domains) {
        if (!isValidDomain(domain)) continue;
        origins.push(new RegExp(`^http(s)?://${escapeRegExp(domain)}$`));
    }
    if (siteIp && isValidIP(siteIp)) {
        origins.push(new RegExp(`^http(s)?://${escapeRegExp(siteIp)}(?::[0-9]+)?$`));
    }

    cachedOrigins = origins;
    return origins;
};

/**
 * Checks if a request comes from a safe origin.
 * @param req The request to check
 * @returns True if the request comes from a safe origin
 */
export const isSafeOrigin = (req: Request): boolean => {
    // Allow all on development. This ensures that graphql-generate and other 
    // dev tools work properly.
    if (process.env.NODE_ENV === "development") return true;
    const origins = safeOrigins();
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
