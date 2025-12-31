/**
 * Maps JSON dot-notation paths to line numbers in a JSON string.
 *
 * Given a JSON string and a path like "target.vps.host", returns the line number
 * where that key appears. This enables highlighting specific lines in a code editor
 * based on validation errors that reference JSON paths.
 */

export interface PathLineInfo {
  line: number; // 1-indexed line number
  column: number; // 0-indexed column where key starts
}

export type PathLineMapping = Record<string, PathLineInfo>;

/**
 * Maps all JSON paths in a JSON string to their line numbers.
 *
 * @param jsonString - The JSON string to parse
 * @returns A mapping of dot-notation paths to line info
 *
 * @example
 * const json = `{
 *   "target": {
 *     "vps": {
 *       "host": ""
 *     }
 *   }
 * }`;
 * const mapping = mapJsonPathsToLines(json);
 * // mapping["target.vps.host"] = { line: 4, column: 6 }
 */
export function mapJsonPathsToLines(jsonString: string): PathLineMapping {
  const mapping: PathLineMapping = {};
  const lines = jsonString.split("\n");

  // Stack to track the current path through the JSON structure
  const pathStack: string[] = [];
  // Stack to track if we're in an array at each depth (for array index tracking)
  const arrayStack: boolean[] = [];
  // Stack to track array indices at each depth
  const indexStack: number[] = [];

  // Track nesting depth
  let depth = 0;

  // Regex to match a JSON key
  const keyPattern = /^\s*"([^"\\]*(?:\\.[^"\\]*)*)"\s*:/;

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const line = lines[lineIndex];
    const lineNum = lineIndex + 1;

    // Check for key-value pair
    const keyMatch = line.match(keyPattern);
    if (keyMatch) {
      const key = keyMatch[1];
      const column = line.indexOf('"');

      // Count braces/brackets before the key to determine depth adjustment
      const beforeKey = line.slice(0, column);
      const openBraces = (beforeKey.match(/\{/g) || []).length;
      const closeBraces = (beforeKey.match(/\}/g) || []).length;
      const openBrackets = (beforeKey.match(/\[/g) || []).length;
      const closeBrackets = (beforeKey.match(/\]/g) || []).length;

      // Adjust path stack based on depth changes
      depth += openBraces + openBrackets - closeBraces - closeBrackets;

      // Pop path stack to match current depth
      while (pathStack.length >= depth) {
        pathStack.pop();
        if (arrayStack.length > pathStack.length) {
          arrayStack.pop();
          indexStack.pop();
        }
      }

      // Add current key to path
      pathStack.push(key);

      // Record the path
      const fullPath = pathStack.join(".");
      mapping[fullPath] = { line: lineNum, column };

      // Check if value starts an object or array (on the same line after the key)
      const afterKey = line.slice(line.indexOf(":") + 1);
      if (afterKey.includes("{")) {
        depth++;
        arrayStack.push(false);
        indexStack.push(0);
      } else if (afterKey.includes("[")) {
        depth++;
        arrayStack.push(true);
        indexStack.push(0);
      }
    } else {
      // No key on this line - track structure changes
      const openBraces = (line.match(/\{/g) || []).length;
      const closeBraces = (line.match(/\}/g) || []).length;
      const openBrackets = (line.match(/\[/g) || []).length;
      const closeBrackets = (line.match(/\]/g) || []).length;

      // Handle closing braces/brackets - pop from path stack
      for (let i = 0; i < closeBraces + closeBrackets; i++) {
        if (pathStack.length > 0) {
          pathStack.pop();
        }
        if (arrayStack.length > 0) {
          arrayStack.pop();
          indexStack.pop();
        }
        depth--;
      }

      // Handle opening braces/brackets
      depth += openBraces + openBrackets;
    }
  }

  return mapping;
}

/**
 * Gets the line number for a specific JSON path.
 *
 * @param jsonString - The JSON string to parse
 * @param path - The dot-notation path to find
 * @returns The line number (1-indexed) or null if not found
 */
export function getLineForPath(jsonString: string, path: string): number | null {
  const mapping = mapJsonPathsToLines(jsonString);
  return mapping[path]?.line ?? null;
}

/**
 * Gets line info for multiple paths at once (more efficient than calling getLineForPath multiple times).
 *
 * @param jsonString - The JSON string to parse
 * @param paths - Array of dot-notation paths to find
 * @returns A mapping of paths to line numbers (only includes found paths)
 */
export function getLinesForPaths(
  jsonString: string,
  paths: string[]
): Record<string, number> {
  const mapping = mapJsonPathsToLines(jsonString);
  const result: Record<string, number> = {};

  for (const path of paths) {
    if (mapping[path]) {
      result[path] = mapping[path].line;
    }
  }

  return result;
}
