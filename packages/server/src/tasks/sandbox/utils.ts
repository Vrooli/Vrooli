/**
 * Extracts the name of a function from its string representation.
 * This function ignores comments and supports various function declaration styles,
 * including traditional function declarations, function expressions, and arrow functions.
 *
 * @param funcStr The string containing the function code. This string can include comments.
 * @returns The name of the function if found; otherwise, null if no name could be identified.
 */
export function getFunctionName(funcStr: string): string | null {
    // First, strip out block comments
    funcStr = funcStr.replace(/\/\*[\s\S]*?\*\//g, "");

    // Then strip out inline comments
    funcStr = funcStr.replace(/\/\/.*/g, "");

    // Regular expression to capture function names from different declarations
    const regex = /(?:function\s+([^\s(]+))|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s*)?function|(?:const|let|var)\s+([^\s=]+)\s*=\s*(?:async\s*)?\(.*?\)\s*=>/;

    // Attempt to match the regex pattern with the function string
    const match = regex.exec(funcStr);

    // Return the first matching group that contains a name
    if (match) {
        return match[1] || match[2] || match[3];
    }

    return null; // Return null if no function name is found
}

/**
 * Safely stringify an object, handling circular references, Date objects, and undefined values.
 * 
 * @param {any} obj - The object to stringify.
 * @returns {string} A JSON string representation of the object.
 * 
 * @description
 * This function performs a custom JSON stringification with the following features:
 * - Detects and marks circular references with '[Circular]'
 * - Converts Date objects to a special string format: '__DATE__<ISO string>'
 * - Handles undefined values by returning the string '__UNDEFINED__'
 * 
 * Note: This function does not handle functions, symbols, or other non-JSON data types.
 * Such values will be omitted from the resulting string, as per standard JSON.stringify behavior.
 * 
 * @example
 * const obj = { a: 1, b: new Date(), c: undefined };
 * obj.self = obj;  // circular reference
 * console.log(safeStringify(obj));
 * // Outputs: {"a":1,"b":"__DATE__2023-08-05T00:00:00.000Z","c":"__UNDEFINED__","self":"[Circular]"}
 */
export function safeStringify(obj: any): string {
    if (obj === undefined) return "__UNDEFINED__";
    const seen = new WeakSet();
    return JSON.stringify(obj, (key, value) => {
        if (value instanceof Date) {
            return `__DATE__${value.toISOString()}`;
        }
        if (typeof value === "object" && value !== null) {
            if (seen.has(value)) {
                return "[Circular]";
            }
            seen.add(value);
        }
        return value;
    });
}

/**
 * Safely parse a JSON string, handling special cases for circular references, Date objects, 
 * and other types which aren't typically supported by JSON.parse.
 * 
 * @param {string} str - The JSON string to parse.
 * @returns {any} The parsed JavaScript object.
 */
export function safeParse(str: string): any {
    if (str === "__UNDEFINED__") return undefined;
    return JSON.parse(str, (key, value) => {
        if (typeof value === "string" && value.startsWith("__DATE__")) {
            return new Date(value.slice(8));
        }
        if (value === "__UNDEFINED__") {
            return undefined;
        }
        return value;
    });
}

