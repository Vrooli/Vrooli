/**
 * Converts url search params to object.
 * See https://stackoverflow.com/a/8649003/10240279
 * @param searchParams Search params (i.e. window.location.search)
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (searchParams: string): { [key: string]: string } => {
    if (searchParams.length <= 1 || !searchParams.startsWith('?')) return {};
    const search = searchParams.substring(1);
    try {
        return JSON.parse('{"' + decodeURI(search).replace(/"/g, '\\"').replace(/&/g, '","').replace(/=/g, '":"') + '"}')
    } catch(error) {
        console.error('caught error in parseSearchParams', error);
        return {};
    }
}

/**
 * Converts object to url search params.
 * @param params Object with key/value pairs, representing search params
 * @returns string of search params, matching the format of window.location.search
 */
export const stringifySearchParams = (params: { [key: string]: string }): string => {
    const keys = Object.keys(params);
    if (keys.length === 0) return '';
    const encodedParams = keys.map(key => encodeURIComponent(key) + '=' + encodeURIComponent(params[key])).join('&');
    return '?' + encodedParams;
}