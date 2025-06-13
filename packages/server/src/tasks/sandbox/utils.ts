// Import CustomTransformer type directly from superjson
import type { SuperJSONValue } from "superjson";

// Define CustomTransformer type locally if needed
type CustomTransfomer<T = any> = {
    name: string;
    isApplicable: (v: any) => v is T;
    serialize: (v: T) => SuperJSONValue;
    deserialize: (v: SuperJSONValue) => T;
};

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

    // Remove leading and trailing whitespace
    funcStr = funcStr.trim();

    // Regular expression to capture function names from different declarations
    // const regex = /(?:function\s+([^\s(]+))|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s*)?function|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s*)?\(.*?\)\s*=>/;
    const regex = /(?:async\s+)?function(?:\s*\*)?\s+([^\s(]+)|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s+)?function(?:\s*\*)?|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[^\s=]+)\s*=>/;

    // Attempt to match the regex pattern with the function string
    const match = regex.exec(funcStr);
    if (match) {
        const functionName = (match[1] || match[2] || match[3]).trim();
        const isAsync = match[0]?.startsWith("async ") || match[0]?.includes(" async ");
        return {
            // Return the first matching group that contains a name
            functionName,
            // Check the matched group to see if 'async' is present
            isAsync,
        };
    }

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
        // Add more properties as needed
    };
}

/**
 * @returns A wrapper for creating a Buffer object, which is used in
 * conjunction with a custom Buffer class to inject the Buffer class into the isolate.
 */
export function bufferWrapper(data: string, encoding?: BufferEncoding) {
    const bufferObject = Buffer.from(data, encoding);
    return {
        byteLength: bufferObject.byteLength,
        length: bufferObject.length,
    };
}

/**
 * Custom register to handle URL objects in the isolate.
 */
export const urlRegister: Omit<CustomTransfomer<URL, string>, "name"> = {
    isApplicable(value: unknown): value is URL {
        return value instanceof URL;
    },
    serialize: (value: URL) => value.href,
    deserialize: (value: string) => urlWrapper(value) as URL,
};

/**
 * Custom register to handle Buffer objects in the isolate.
 */
export const bufferRegister: Omit<CustomTransfomer<Buffer, string>, "name"> = {
    isApplicable(value: unknown): value is Buffer {
        return value instanceof Buffer;
    },
    serialize: (value: Buffer) => value.toString("base64"),
    deserialize: (value: string) => Buffer.from(value, "base64"),
};

/**
 * Creates the URL class in the isolate.
 */
export function getURLClassString(urlId: string) {
    return `
   class URL {
        constructor(url, base) {
            const urlData = ${urlId}(url, base);
            Object.assign(this, urlData);
        }

        toString() {
            return this.href;
        }
    }

    // Make URL available globally
    globalThis.URL = URL;
`;
}

/**
 * Creates the Buffer class in the isolate.
 */
export function getBufferClassString(bufferId: string) {
    return `
    class Buffer {
        constructor(size) {
            const bufferData = ${bufferId}(size);
            Object.assign(this, bufferData);
        }

        toString() {
            return this.size;
        }

        from(array) {
            return new Buffer(array.length);
        }

        static from(array) {
            return new Buffer(array.length);
        }
    }

    // Make Buffer available globally
    globalThis.Buffer = Buffer;
`;
}
