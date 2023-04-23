export const safeOrigins = () => {
    const origins = ["https://cardano-mainnet.blockfrost.io"];
    if (process.env.VITE_SERVER_LOCATION === "local") {
        origins.push(/^http:\/\/localhost(?::[0-9]+)?$/, /^http:\/\/192.168.0.[0-9]{1,2}(?::[0-9]+)?$/, "https://studio.apollographql.com", new RegExp(`^http(s)?:\/\/${process.env.VITE_SITE_IP}(?::[0-9]+)?$`));
    }
    else {
        const domains = (process.env.VIRTUAL_HOST ?? "").split(",");
        for (const domain of domains) {
            origins.push(new RegExp(`^http(s)?:\/\/${domain}$`));
        }
        origins.push(new RegExp(`^http(s)?:\/\/${process.env.VITE_SITE_IP}(?::[0-9]+)?$`));
    }
    return origins;
};
export const isSafeOrigin = (req) => {
    if (process.env.NODE_ENV === "development")
        return true;
    const origins = safeOrigins();
    const origin = req.headers.origin;
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
//# sourceMappingURL=origin.js.map