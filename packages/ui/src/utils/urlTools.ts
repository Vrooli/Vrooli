/**
 * Converts url search params to object.
 * See https://stackoverflow.com/a/8649003/10240279
 * @param searchParams Search params (i.e. window.location.search)
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (searchParams: string): { [key: string]: string } => {
    if (searchParams.length <= 1 || !searchParams.startsWith('?')) return {};
    const search = searchParams.substring(1);
    return JSON.parse('{"' + decodeURI(search).replace(/"/g, '\\"').replace(/&/g, '","').replace(/=/g, '":"') + '"}')
}