export const yupFields = (schema, recurseBase = "") => {
    const isRequired = recurseBase.length === 0 || Boolean(schema.tests.find(test => test.OPTIONS.name === "required"));
    const prepend = recurseBase.length > 0 ? recurseBase + (isRequired ? "" : "?") + "." : "";
    if (schema.type === "object") {
        const required = [];
        for (const [key, value] of Object.entries(schema.fields)) {
            console.log("object required recurse", key, prepend, value);
            required.push(...yupFields(value, prepend + key));
        }
        return required;
    }
    else if (schema.type === "array") {
        console.log("array required recurse", prepend, schema);
        return yupFields(schema.innerType, recurseBase + (isRequired ? "" : "?"));
    }
    else {
        console.log("in else", recurseBase, isRequired);
        return isRequired ? [recurseBase] : [recurseBase + "?"];
    }
};
export const yupObjectContainsRequiredFields = (object, fields) => {
    if (object === undefined || object === null || typeof object !== "object")
        return false;
    const topLevelRequired = fields.map(field => field.split(".")[0]).filter(field => field.endsWith("?")).map(field => field.slice(0, -1));
    if (!topLevelRequired.every(field => object[field] !== undefined))
        return false;
    const nextLevel = fields.map(field => field.split(".").slice(1).join(".")).filter(field => field.length > 0);
    for (const [key, value] of Object.entries(object)) {
        const nextLevelFields = nextLevel.filter(field => field.startsWith(key + ".") || field.startsWith(key + "?."));
        if (nextLevelFields.length > 0) {
            if (!yupObjectContainsRequiredFields(value, nextLevelFields))
                return false;
        }
    }
    return true;
};
export const grabValidTopLevelFields = (object, fields) => {
    console.log("grabValidTopLevelFields", object, fields);
    if (object === undefined || object === null || typeof object !== "object")
        return {};
    const topLevel = fields.map(field => field.split(".")[0]).map(field => field.endsWith("?") ? field.slice(0, -1) : field);
    const returnObject = {};
    for (const field of topLevel) {
        if (object[field] !== undefined)
            returnObject[field] = object[field];
    }
    return returnObject;
};
export const isYupValidationError = (error) => {
    return error.name === "ValidationError";
};
export const validateAndGetYupErrors = async (schema, values) => {
    try {
        await schema.validate(values);
        return {};
    }
    catch (error) {
        if (isYupValidationError(error)) {
            if (error.inner.length > 0) {
                return error.inner.reduce((errors, err) => {
                    if (err && err.path) {
                        errors[err.path] = err.message;
                    }
                    return errors;
                }, {});
            }
            else if (error.path) {
                return { [error.path]: error.message };
            }
        }
        throw error;
    }
};
//# sourceMappingURL=yupTools.js.map