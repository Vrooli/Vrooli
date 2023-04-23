import { exists, isObject } from "@local/utils";
import fs from "fs";
import pkg from "lodash";
import { CustomError } from "../events/error";
const { flatten } = pkg;
const profanity = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/dist/utils/censorDictionary.txt`).toString().split("\n");
const profanityRegex = new RegExp(profanity.map(word => `(?=\\b)${word}(?=\\b)`).join("|"), "gi");
export const hasProfanity = (...text) => {
    return text.some(t => exists(t) && t.search(profanityRegex) !== -1);
};
export const toStringArray = (item, fields) => {
    if (Array.isArray(item)) {
        return flatten(item.map(i => toStringArray(i, fields))).filter(i => i !== null);
    }
    else if (isObject(item) && Object.prototype.toString.call(item) !== "[object Date]") {
        const childFields = fields ? fields.map(s => s.split(".").slice(1).join(".")).filter(s => s.length > 0) : null;
        const valuesToCheck = [];
        for (const [key, value] of Object.entries(item)) {
            if (fields && !fields.includes(key))
                continue;
            valuesToCheck.push(value);
        }
        return flatten(valuesToCheck.map(i => toStringArray(i, childFields))).filter(i => i !== null);
    }
    else if (typeof item === "string") {
        return [item];
    }
    return null;
};
export const validateProfanity = (items, languages) => {
    const strings = items.filter(i => typeof i === "string");
    if (hasProfanity(...strings))
        throw new CustomError("0042", "BannedWord", languages);
};
export const filterProfanity = (text, censorCharacter = "*") => {
    return text.replace(profanityRegex, (s) => {
        let i = 0;
        let asterisks = "";
        while (i < s.length) {
            asterisks += censorCharacter;
            i++;
        }
        return asterisks;
    });
};
//# sourceMappingURL=censor.js.map