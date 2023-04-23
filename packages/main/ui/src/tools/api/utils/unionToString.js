import { partialToStringHelper } from "./partialToStringHelper";
export const unionToString = async (union, indent = 0) => {
    let result = "";
    for (const [key, value] of Object.entries(union)) {
        result += " ".repeat(indent);
        result += `... on ${key} {\n`;
        if (typeof value === "string") {
            result += `${" ".repeat(indent + 4)}...${value}\n`;
        }
        else if (typeof value === "object" || typeof value === "function") {
            result += await partialToStringHelper(typeof value === "function" ? await value() : value, indent + 4);
        }
        else {
            console.error("unionToString got unexpected value", key, value);
        }
        result += `${" ".repeat(indent)}}\n`;
    }
    return result;
};
//# sourceMappingURL=unionToString.js.map