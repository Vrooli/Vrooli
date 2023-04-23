import { InputType } from "@local/consts";
const radioValueToSearch = (value) => {
    if (value === "true" || value === "false")
        return value === "true";
    if (value === "undefined")
        return undefined;
    return value;
};
const searchToRadioValue = (value) => value + "";
function arrayConvert(arr, convert) {
    if (Array.isArray(arr)) {
        if (arr.length === 0)
            return undefined;
        return convert ? arr.map(convert) : arr;
    }
    return undefined;
}
;
const inputTypeToSearch = {
    [InputType.Checkbox]: (value) => value,
    [InputType.Dropzone]: (value) => value,
    [InputType.IntegerInput]: (value) => value,
    [InputType.JSON]: (value) => value,
    [InputType.LanguageInput]: (value) => arrayConvert(value),
    [InputType.Markdown]: (value) => typeof value === "string" && value.trim().length > 0 ? value.trim() : undefined,
    [InputType.Prompt]: (value) => typeof value === "string" && value.trim().length > 0 ? value.trim() : undefined,
    [InputType.Radio]: (value) => radioValueToSearch(value),
    [InputType.Selector]: (value) => value,
    [InputType.Slider]: (value) => value,
    [InputType.Switch]: (value) => value,
    [InputType.TagSelector]: (value) => arrayConvert(value, ({ tag }) => tag),
    [InputType.TextField]: (value) => typeof value === "string" && value.trim().length > 0 ? value.trim() : undefined,
};
const searchToInputType = {
    [InputType.Checkbox]: (value) => value,
    [InputType.Dropzone]: (value) => value,
    [InputType.IntegerInput]: (value) => value,
    [InputType.JSON]: (value) => value,
    [InputType.LanguageInput]: (value) => arrayConvert(value),
    [InputType.Markdown]: (value) => value,
    [InputType.Prompt]: (value) => value,
    [InputType.Radio]: (value) => searchToRadioValue(value),
    [InputType.Selector]: (value) => value,
    [InputType.Slider]: (value) => value,
    [InputType.Switch]: (value) => value,
    [InputType.TagSelector]: (value) => arrayConvert(value, (tag) => ({ tag })),
    [InputType.TextField]: (value) => value,
};
export const convertFormikForSearch = (values, schema) => {
    const result = {};
    for (const field of schema.fields) {
        if (values[field.fieldName]) {
            const value = inputTypeToSearch[field.type](values[field.fieldName]);
            if (value !== undefined)
                result[field.fieldName] = value;
        }
    }
    return result;
};
export const convertSearchForFormik = (values, schema) => {
    const result = {};
    for (const field of schema.fields) {
        if (values[field.fieldName]) {
            const value = searchToInputType[field.type](values[field.fieldName]);
            result[field.fieldName] = value;
        }
        else
            result[field.fieldName] = undefined;
    }
    return result;
};
//# sourceMappingURL=inputToSearch.js.map