import { isRelationshipArray, isRelationshipObject } from "../builders";
import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { hasProfanity } from "../utils/censor";
const collectProfanities = (input, objectType) => {
    const result = {};
    const format = objectType ? ObjectMap[objectType]?.format : undefined;
    const validate = objectType ? ObjectMap[objectType]?.validate : undefined;
    if (validate?.profanityFields) {
        for (const field of validate.profanityFields) {
            if (input[field])
                result[field] = result[field] ? [...result[field], input[field]] : [input[field]];
        }
    }
    if (isRelationshipArray(input.translationsCreate) && input.translationsCreate.length > 0) {
        for (const field of Object.keys(input.translationsCreate[0])) {
            if (field !== "id" && !field.endsWith("Id")) {
                const values = input.translationsCreate.map((x) => x[field]);
                result[field] = result[field] ? [...result[field], ...values] : values;
            }
        }
    }
    if (input.translationsUpdate) {
        for (const field of Object.keys(input.translationsUpdate)) {
            if (field !== "id" && !field.endsWith("Id")) {
                const values = input.translationsUpdate.map((x) => x[field]);
                result[field] = result[field] ? [...result[field], ...values] : values;
            }
        }
    }
    for (const key in input) {
        let nextObjectType;
        const strippedKey = key.endsWith("Create") || key.endsWith("Update") ? key.slice(0, -6) : key;
        if (typeof format?.gqlRelMap?.[strippedKey] === "string")
            nextObjectType = format?.gqlRelMap?.[strippedKey];
        if (isRelationshipArray(input[key])) {
            for (const item of input[key]) {
                const newFields = collectProfanities(item, nextObjectType);
                for (const field in newFields) {
                    result[field] = result[field] ? [...result[field], ...newFields[field]] : newFields[field];
                }
            }
        }
        else if (isRelationshipObject(input[key])) {
            const newFields = collectProfanities(input[key], nextObjectType);
            for (const field in newFields) {
                result[field] = result[field] ? [...result[field], ...newFields[field]] : newFields[field];
            }
        }
    }
    return result;
};
export const profanityCheck = (input, objectType, languages) => {
    const fieldsToCheck = {};
    for (const item of input) {
        const newFields = collectProfanities(item, objectType);
        for (const field in newFields) {
            fieldsToCheck[field] = fieldsToCheck[field] ? [...fieldsToCheck[field], ...newFields[field]] : newFields[field];
        }
    }
    for (const field in fieldsToCheck) {
        if (hasProfanity(...fieldsToCheck[field])) {
            throw new CustomError("0115", "BannedWord", languages);
        }
    }
};
//# sourceMappingURL=profanityCheck.js.map