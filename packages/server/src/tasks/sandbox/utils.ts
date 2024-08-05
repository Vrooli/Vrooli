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
