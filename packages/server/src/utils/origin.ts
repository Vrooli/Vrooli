import { Request } from "express";

/**
 * Origins which are allowed to make requests without an API key.
 * @returns An array of strings and regular expressions
 */
export const safeOrigins = (): Array<string | RegExp> => {
    const origins: Array<string | RegExp> = ["https://cardano-mainnet.blockfrost.io"];
    if (process.env.VITE_SERVER_LOCATION === "local") {
        origins.push(
            /^http:\/\/localhost(?::[0-9]+)?$/,
            /^http:\/\/192.168.0.[0-9]{1,2}(?::[0-9]+)?$/,
            "https://studio.apollographql.com",
            new RegExp(`^http(s)?:\/\/${process.env.SITE_IP}(?::[0-9]+)?$`),
        );
    }
    else {
        // Parse URLs from process.env.VIRTUAL_HOST
        const domains = (process.env.VIRTUAL_HOST ?? "").split(",");
        for (const domain of domains) {
            origins.push(new RegExp(`^http(s)?:\/\/${domain}$`));
        }
        origins.push(
            new RegExp(`^http(s)?:\/\/${process.env.SITE_IP}(?::[0-9]+)?$`),
        );
    }
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
