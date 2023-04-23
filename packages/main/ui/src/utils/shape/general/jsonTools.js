import { isObject } from "@local/utils";
export const jsonHelpText = `JSON is a widely used format for storing data objects.  

If you are unfamiliar with JSON, you may [read this guide](https://www.tutorialspoint.com/json/json_quick_guide.htm) to learn how to use it.  

On top of the JSON format, we also support the following:  
- **Variables**: Any key or value can be substituted with a variable. These are used to make it easy for users to fill in their own data, as well as 
provide details about those parts of the JSON. Variables follow the format of &lt;variable_name&gt;.  
- **Optional fields**: Fields that are not required can be marked as optional. Optional fields must start with a question mark (e.g. ?field_name).  
- **Additional fields**: If an object contains arbitrary fields, add a field with brackets (e.g. [variable]).  
- **Data types**: Data types are specified by reserved words (e.g. &lt;string&gt;, &lt;number | boolean&gt;, &lt;any&gt;, etc.), variable names (e.g. &lt;variable_name&gt;), standard IDs (e.g. &lt;decdf6c8-4230-4777-b8e3-799df2503c42&gt;), or simply entered as normal JSON.
`;
const isActualObject = (obj) => !Array.isArray(obj) && isObject(obj) && Object.prototype.toString.call(obj) !== "[object Date]";
export const isJson = (value) => {
    try {
        if (typeof value === "object")
            return true;
        value && JSON.parse(value);
        return true;
    }
    catch (e) {
        return false;
    }
};
export const compareJSONHelper = (fieldName, path, format, data) => {
    return {};
    if (Array.isArray(format)) {
        if (!Array.isArray(data))
            return [{
                    fieldName,
                    path,
                    message: `Expected array, got ${typeof data}`,
                }];
    }
    else if (isActualObject(format)) {
        if (!isActualObject(data))
            return [{
                    fieldName,
                    path,
                    message: `Expected object, got ${typeof data}`,
                }];
    }
    else if (typeof format === "string" && (format.startsWith("<") || format.startsWith("?<")) && format.endsWith(">")) {
    }
    else if (typeof format === "string" && format.startsWith("[") && format.endsWith("]")) {
    }
    else {
        if (Array.isArray(data) || isActualObject(data))
            return [{
                    fieldName,
                    path,
                    message: `Expected ${format}, got ${typeof data}`,
                }];
    }
};
export const compareJSONToFormat = (format, data) => {
    let formatJSON;
    let dataJSON;
    try {
        formatJSON = JSON.parse(format);
        dataJSON = JSON.parse(data);
        const errors = compareJSONHelper("$root", [], formatJSON, dataJSON);
        return {
            isValid: errors.length === 0,
            errors,
        };
    }
    catch (error) {
        console.error("Error parsing JSON format or data", error);
        return {
            isValid: false,
            errors: [{
                    fieldName: "$root",
                    path: [],
                    message: "Failed to parse JSON format or data",
                }],
        };
    }
};
export const findVariablePositions = (variable, format) => {
    const variablePositions = [];
    if (!format)
        return variablePositions;
    const formatString = JSON.stringify(format);
    const keySubstring = `"<${variable}>":`;
    let openBracketCounter = 0;
    let inQuotes = false;
    let index = 1;
    let keyStartIndex = -1;
    let valueStartIndex = -1;
    while (index < formatString.length) {
        const currChar = formatString[index];
        const lastChar = formatString[index - 1];
        if (currChar !== "\\" && lastChar !== "\\") {
            if (keyStartIndex === -1) {
                if (formatString.length - index >= keySubstring.length && formatString.substring(index, index + keySubstring.length) === keySubstring) {
                    keyStartIndex = index;
                }
            }
            else if (index === keyStartIndex + keySubstring.length) {
                if (currChar !== "\"" && currChar !== "{") {
                    keyStartIndex = -1;
                    valueStartIndex = -1;
                }
                valueStartIndex = index;
                openBracketCounter = Number(currChar === "{");
                inQuotes = currChar === "\"";
            }
            else if (index > keyStartIndex + keySubstring.length) {
                openBracketCounter += Number(currChar === "{");
                openBracketCounter -= Number(currChar === "}");
                if (currChar === "\"")
                    inQuotes = !inQuotes;
                if (openBracketCounter === 0 && !inQuotes) {
                    variablePositions.push({
                        field: {
                            start: keyStartIndex,
                            end: keyStartIndex + keySubstring.length,
                        },
                        value: {
                            start: valueStartIndex,
                            end: index,
                        },
                    });
                }
            }
        }
        else
            index++;
        index++;
    }
    return variablePositions;
};
export const jsonToMarkdown = (value) => {
    try {
        const parsed = typeof value === "string" ? JSON.parse(value) : value;
        return value ? `\`\`\`json\n${JSON.stringify(parsed, null, 2)}\n\`\`\`` : null;
    }
    catch (e) {
        return null;
    }
};
export const jsonToString = (value) => {
    try {
        const parsed = typeof value === "string" ? JSON.parse(value) : value;
        return value ? JSON.stringify(parsed, null, 2) : null;
    }
    catch (e) {
        return null;
    }
};
export const isEqualJSON = (a, b) => {
    try {
        const parsedA = typeof a === "string" ? JSON.parse(a) : a;
        const parsedB = typeof b === "string" ? JSON.parse(b) : b;
        return JSON.stringify(parsedA) === JSON.stringify(parsedB);
    }
    catch (e) {
        return false;
    }
};
export const safeStringify = (value) => {
    if (typeof value === "string" && value.length > 1 && value[0] === "{" && value[value.length - 1] === "}") {
        return value;
    }
    return JSON.stringify(value);
};
//# sourceMappingURL=jsonTools.js.map