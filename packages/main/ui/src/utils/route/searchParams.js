export const parseSearchParams = () => {
    const searchParams = window.location.search;
    if (searchParams.length <= 1 || !searchParams.startsWith("?"))
        return {};
    let search = searchParams.substring(1);
    try {
        search = search.replace(/([^&=]+)=([^&]*)/g, (match, key, value) => {
            if (value.startsWith("\"") || value.includes("%") || value === "true" || value === "false")
                return match;
            return `${key}="${value}"`;
        });
        const parsed = JSON.parse("{\""
            + decodeURI(search)
                .replace(/&/g, ",\"")
                .replace(/=/g, "\":")
                .replace(/%2F/g, "/")
                .replace(/%5B/g, "[")
                .replace(/%5D/g, "]")
                .replace(/%5C/g, "\\")
                .replace(/%2C/g, ",")
                .replace(/%3A/g, ":")
            + "}");
        Object.keys(parsed).forEach((key) => {
            const value = parsed[key];
            if (typeof value === "string" && value.startsWith("{") && value.endsWith("}")) {
                try {
                    parsed[key] = JSON.parse(value);
                }
                catch (e) {
                }
            }
        });
        return parsed;
    }
    catch (error) {
        console.error("Could not parse search params", error);
        return {};
    }
};
export const stringifySearchParams = (params) => {
    const keys = Object.keys(params);
    if (keys.length === 0)
        return "";
    const filteredKeys = keys.filter(key => params[key] !== null && params[key] !== undefined);
    const encodedParams = filteredKeys.map(key => encodeURIComponent(key) + "=" + encodeURIComponent(JSON.stringify(params[key]))).join("&");
    return "?" + encodedParams;
};
export const addSearchParams = (setLocation, params) => {
    const currentParams = parseSearchParams();
    const newParams = { ...currentParams, ...params };
    setLocation(window.location.pathname, { replace: true, searchParams: newParams });
};
export const setSearchParams = (setLocation, params) => {
    setLocation(window.location.pathname, { replace: true, searchParams: params });
};
export const keepSearchParams = (setLocation, keep) => {
    const keepResult = {};
    const searchParams = parseSearchParams();
    keep.forEach(key => {
        if (searchParams[key] !== undefined)
            keepResult[key] = searchParams[key];
    });
    setLocation(window.location.pathname, { replace: true, searchParams: keepResult });
};
export const removeSearchParams = (setLocation, remove) => {
    const removeResult = {};
    const searchParams = parseSearchParams();
    Object.keys(searchParams).forEach(key => {
        if (!remove.includes(key))
            removeResult[key] = searchParams[key];
    });
    setLocation(window.location.pathname, { replace: true, searchParams: removeResult });
};
//# sourceMappingURL=searchParams.js.map