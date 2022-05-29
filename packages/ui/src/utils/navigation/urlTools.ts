type Primitive = string | number | boolean;
type ParseSearchParamsResult = { [x: string]: Primitive | Primitive[] | ParseSearchParamsResult };

/**
 * Converts url search params to object
 * See https://stackoverflow.com/a/8649003/10240279
 * @param searchParams Search params (i.e. window.location.search)
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (searchParams: string): ParseSearchParamsResult => {
    if (searchParams.length <= 1 || !searchParams.startsWith('?')) return {};
    const search = searchParams.substring(1);
    try {
        // Decode and parse the search params
        const parsed = JSON.parse('{"'
        + decodeURI(search)
            .replace(/&/g, ',"')
            .replace(/=/g, '":')
            .replace(/%2F/g, '/')
            .replace(/%5B/g, '[')
            .replace(/%5D/g, ']')
            .replace(/%5C/g, '\\')
            .replace(/%2C/g, ',')
            .replace(/%3A/g, ':')
        + '}');
        // For any value that might be JSON, try to parse
        Object.keys(parsed).forEach((key) => {
            const value = parsed[key];
            if (typeof value === 'string' && value.startsWith('{') && value.endsWith('}')) {
                try {
                    parsed[key] = JSON.parse(value);
                } catch (e) {
                    // Do nothing
                }
            }
        });
        return parsed;
    } catch (error) {
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
    console.log('qq filtered keyssssss', filteredKeys);
    const temp = filteredKeys.map(key => {
        console.log('qq key', key);
        console.log('qq params[key]', params[key]);
        console.log('qq encodedkey', encodeURIComponent(key));
        console.log('qq encodedvalue 1', encodeURIComponent(params[key]));
        console.log('qq encodedvalue 2', encodeURIComponent(JSON.stringify(params[key])));
        return encodeURIComponent(key) + '=' + encodeURIComponent(JSON.stringify(params[key]))
    })
    console.log('qq filtered keysssssss resullttttttt', temp);
    console.log('qq & combine', temp.join('&'));
    const encodedParams = filteredKeys.map(key => encodeURIComponent(key) + '=' + encodeURIComponent(JSON.stringify(params[key]))).join('&');
    return '?' + encodedParams;
}