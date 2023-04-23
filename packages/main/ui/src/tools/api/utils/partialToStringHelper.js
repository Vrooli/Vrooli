import { unionToString } from "./unionToString";
export const partialToStringHelper = async (partial, indent = 0) => {
    if (indent > 69) {
        throw new Error("partialToStringHelper indent too high. Possible infinite loop.");
    }
    let result = "";
    for (const [key, value] of Object.entries(partial)) {
        Array.isArray(value) && console.error("Array value in partialToStringHelper", key, value);
        if (["__typename", "__selectionType", "__define"].includes(key))
            continue;
        else if (key === "__union") {
            result += await unionToString(value, indent);
        }
        else if (key === "__use") {
            result += `${" ".repeat(indent)}...${value}\n`;
        }
        else if (typeof value === "boolean") {
            result += `${" ".repeat(indent)}${key}\n`;
        }
        else {
            result += `${" ".repeat(indent)}${key} {\n`;
            result += await partialToStringHelper(typeof value === "function" ? await value() : value, indent + 4);
            result += `${" ".repeat(indent)}}\n`;
        }
    }
    return result;
};
//# sourceMappingURL=partialToStringHelper.js.map