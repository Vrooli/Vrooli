import { logger } from "../events";

export const objectToYaml = (obj: Record<string, any>, indentLevel: number = 0) => {
    if (obj === null || obj === undefined || typeof obj !== 'object' || Array.isArray(obj)) {
        logger.error('Invalid input to objectToYaml', { trace: "0051", obj });
        return "";
    }

    let yaml = "";
    const indent = " ".repeat(indentLevel);

    for (const [key, value] of Object.entries(obj)) {
        if (typeof value === "object" && !Array.isArray(value)) {
            yaml += `${indent}${key}:\n${objectToYaml(value, indentLevel + 2)}`;
        } else if (Array.isArray(value)) {
            yaml += `${indent}${key}:\n${arrayToYaml(value, indentLevel + 2)}`;
        } else {
            // Assuming all values are strings or can be safely converted to strings
            yaml += `${indent}${key}: ${value}\n`;
        }
    }

    return yaml;
}

export const arrayToYaml = (arr: Array<any>, indentLevel: number = 0) => {
    if (arr === null || arr === undefined || !Array.isArray(arr)) {
        logger.error('Invalid input to arrayToYaml', { trace: "0053", arr });
        return "";
    }

    let yaml = "";
    const indent = " ".repeat(indentLevel);

    for (const item of arr) {
        if (typeof item === 'object' && !Array.isArray(item)) {
            const itemEntries = Object.entries(item).map(([key, value]) =>
                `${indent}- ${key}: ${value}`).join('\n');
            yaml += `${itemEntries}\n`;
        } else if (Array.isArray(item)) {
            const nestedArrayYaml = arrayToYaml(item, indentLevel + 2);
            yaml += `${indent}-\n${nestedArrayYaml}`;
        } else {
            yaml += `${indent}- ${item}\n`;
        }
    }

    return yaml;
}