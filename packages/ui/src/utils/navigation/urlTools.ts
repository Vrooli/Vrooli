type SearchValue = string | number | boolean | (string | number | boolean)[];
/**
 * Converts url search params to object
 * See https://stackoverflow.com/a/8649003/10240279
 * @param searchParams Search params (i.e. window.location.search)
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (searchParams: string): { [key: string]: SearchValue } => {
    if (searchParams.length <= 1 || !searchParams.startsWith('?')) return {};
    const search = searchParams.substring(1);
    try {
        return JSON.parse('{"'
            + decodeURI(search)
                .replace(/&/g, ',"')
                .replace(/=/g, '":')
                .replace(/%2F/g, '/')
                .replace(/%5B/g, '[')
                .replace(/%5D/g, ']')
                .replace(/%5C/g, '\\')
                .replace(/%2C/g, ',')
            + '}');
    } catch (error) {
        console.error('caught error in parseSearchParams', search, error);
        return {};
    }
}

/**
 * Converts object to url search params. 
 * Ignores keys with undefined or null values.
 * @param params Object with key/value pairs, representing search params
 * @returns string of search params, matching the format of window.location.search
 */
export const stringifySearchParams = (params: { [key: string]: any }): string => {
    const keys = Object.keys(params);
    if (keys.length === 0) return '';
    // Filter out any keys which are associated with undefined or null values
    const filteredKeys = keys.filter(key => params[key] !== undefined && params[key] !== null);
    const encodedParams = filteredKeys.map(key => encodeURIComponent(key) + '=' + encodeURIComponent(JSON.stringify(params[key]))).join('&');
    return '?' + encodedParams;
}