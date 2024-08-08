/**
 * Extracts the name and asynchrony of a function from its string representation.
 * This function ignores comments and supports various function declaration styles,
 * including traditional, expression, and arrow functions, both synchronous and asynchronous.
 *
 * @param funcStr The string containing the function code. This string can include comments.
 * @returns An object containing the name of the function if found and a flag indicating if it's asynchronous.
 *          Returns null if no name could be identified.
 */
export function getFunctionDetails(funcStr: string): { functionName: string | null, isAsync: boolean } {
    // First, strip out block comments
    funcStr = funcStr.replace(/\/\*[\s\S]*?\*\//g, "");

    // Then strip out inline comments
    funcStr = funcStr.replace(/\/\/.*/g, "");

    // Regular expression to capture function names from different declarations
    const regex = /(?:function\s+([^\s(]+))|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s*)?function|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s*)?\(.*?\)\s*=>/;
    // const regex = /(?:async\s+function\s+([^\s(]+))|(?:const|let|var)\s+([^\s=]+)\s*=\s*(async\s+)?function|(?:const|let|var)\s+([^\s=]+)\s*=\s*(async\s+)?\([^)]*\)\s*=>/;

    // Attempt to match the regex pattern with the function string
    const match = regex.exec(funcStr);
    console.log("got match", match);

    // Return the first matching group that contains a name
    // if (match) {
    //     return match[1] || match[2] || match[3];
    // }
    if (match) {
        console.log("matches", match[0], match[1], match[2], match[3], match[4], match[5]);
        console.log("match input", match.input);
        return {
            functionName: match[1] || match[2] || match[3],
            isAsync: match.input.startsWith("async ") || match.input.includes(" async "),
        };
    }
    // if (match) {
    //     return {
    //         functionName: match[1] || match[2] || match[4], // Capture group for the function name
    //         isAsync: !!match[3] || !!match[5], // Boolean flags to check if 'async' is captured before function declaration
    //     };
    // }

    return { functionName: null, isAsync: false };
}

/**
 * @returns A wrapper for creating a URL object, which is used in 
 * conjunction with a custom URL class to inject the URL class into the isolate.
 */
export function urlWrapper(url: string, base?: string | URL) {
    const urlObject = new URL(url, base);
    return {
        hash: urlObject.hash,
        host: urlObject.host,
        hostname: urlObject.hostname,
        href: urlObject.href,
        origin: urlObject.origin,
        password: urlObject.password,
        pathname: urlObject.pathname,
        port: urlObject.port,
        protocol: urlObject.protocol,
        search: urlObject.search,
        searchParams: urlObject.searchParams,
        username: urlObject.username,
    };
}

/**
 * Custom register to handle URL objects in the isolate.
 */
export const urlRegister = {
    isApplicable: (value: unknown): value is URL => value instanceof URL,
    serialize: (value: URL) => value.href,
    deserialize: (value: string) => urlWrapper(value) as URL,
};

/**
 * @returns A wrapper for creating a Buffer object, which is used in
 * conjunction with a custom Buffer class to inject the Buffer class into the isolate.
 */
export function bufferWrapper(data: string | Buffer | Uint8Array) {
    const buffer = Buffer.from(data);
    return {
        type: "Buffer",
        data: Array.from(buffer),
    };
}

/**
 * Custom register to handle Buffer objects in the isolate.
 */
export const bufferRegister = {
    isApplicable: (value: unknown): value is Buffer => {
        console.log("checking if buffer", value, Buffer.isBuffer(value));
        return Buffer.isBuffer(value) || (
            Object.prototype.hasOwnProperty.call(value, "type") &&
            (value as { type: unknown }).type === "Buffer" &&
            Object.prototype.hasOwnProperty.call(value, "data") &&
            Array.isArray((value as { data: unknown }).data)
        );
    },
    // We return an object instead of the array because the array can be confused with a Uint8Array
    serialize: (value: Buffer) => ({
        type: "Buffer",
        data: Array.from(value),
    }),
    deserialize: (value: { type: string; data: number[] }) => {
        return Buffer.from(value.data);
    },
};
