import { CustomError } from "../events/error";
export const sortObjectKeys = (obj) => {
    if (typeof obj !== "object" || obj === null || Object.prototype.toString.call(obj) !== "[object Date]") {
        return obj;
    }
    const sorted = {};
    Object.keys(obj).sort().forEach(key => {
        sorted[key] = sortObjectKeys(obj[key]);
    });
    return sorted;
};
export const sortify = (stringified, languages) => {
    try {
        const obj = JSON.parse(stringified);
        return JSON.stringify(sortObjectKeys(obj));
    }
    catch (error) {
        throw new CustomError("0210", "InvalidArgs", languages);
    }
};
//# sourceMappingURL=objectTools.js.map