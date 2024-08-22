import { logger } from "../events";

const SPACES_IN_TAB = 2;

/**
 * Converts an object to a YAML string.
 * NOTE: Adds a newline at the end of the YAML string.
 * @param obj The object to convert to YAML
 * @param indentLevel The current level of indentation
 * @returns The YAML string 
 */
export function objectToYaml(obj: Record<string, any>, indentLevel = 0) {
    if (obj === null || obj === undefined || typeof obj !== "object" || Array.isArray(obj)) {
        logger.error("Invalid input to objectToYaml", { trace: "0051", obj });
        return "";
    }

    let yaml = "";
    const indent = " ".repeat(indentLevel);

    for (const [key, value] of Object.entries(obj)) {
        // Skip undefined values entirely
        if (value === undefined) {
            continue;
        }
        // Convert null to a string "null"
        else if (value === null) {
            yaml += `${indent}${key}: null\n`;
        }
        // Recurse for nested objects 
        else if (typeof value === "object" && !Array.isArray(value)) {
            yaml += `${indent}${key}:\n${objectToYaml(value, indentLevel + SPACES_IN_TAB)}`;
        }
        // Recurse each item in an array 
        else if (Array.isArray(value)) {
            yaml += `${indent}${key}:\n${arrayToYaml(value, indentLevel + SPACES_IN_TAB)}`;
        }
        // Assume everything else can be safely converted to a string
        else {
            yaml += `${indent}${key}: ${value}\n`;
        }
    }

    return yaml;
}

/**
 * Converts an array to a YAML string.
 * NOTE: Adds a newline at the end of the YAML string.
 * @param arr The array to convert to YAML
 * @param indentLevel The current level of indentation
 * @returns The YAML string
 */
export function arrayToYaml(arr: Array<any>, indentLevel = 0) {
    if (arr === null || arr === undefined || !Array.isArray(arr)) {
        logger.error("Invalid input to arrayToYaml", { trace: "0053", arr });
        return "";
    }

    let yaml = "";
    const indent = " ".repeat(indentLevel);

    for (const item of arr) {
        if (typeof item === "object" && !Array.isArray(item)) {
            let nestedYaml = objectToYaml(item, indentLevel + SPACES_IN_TAB);
            nestedYaml = nestedYaml.replace(" ".repeat(indentLevel + SPACES_IN_TAB), indent + "- ");
            yaml += nestedYaml;
        } else if (Array.isArray(item)) {
            yaml += `${indent}-\n${arrayToYaml(item, indentLevel + SPACES_IN_TAB)}`;
        } else {
            yaml += `${indent}- ${item}\n`;
        }
    }

    return yaml;
}
