import { InputType } from "@local/consts";
import { buildYup } from "schema-to-yup";
export const InputToYupType = {
    [InputType.JSON]: "string",
    [InputType.IntegerInput]: "number",
    [InputType.Markdown]: "string",
    [InputType.Radio]: "string",
    [InputType.Selector]: "string",
    [InputType.Slider]: "number",
    [InputType.Switch]: "boolean",
    [InputType.TextField]: "string",
};
export const generateYupSchema = (formSchema) => {
    if (!formSchema)
        return null;
    const shape = {
        title: "validationSchema",
        type: "object",
        required: [],
        properties: {},
    };
    const config = { errMessages: {} };
    formSchema.fields.forEach(field => {
        const name = field.fieldName;
        if (field.yup && InputToYupType[field.type]) {
            shape.properties[name] = { type: InputToYupType[field.type] };
            config.errMessages[name] = {};
            if (field.yup.required) {
                shape.properties[name].required = true;
                shape.required.push(name);
                config.errMessages[name].required = `${field.label} is required`;
                config.errMessages[name].string = `${field.label} is required`;
            }
            else {
                shape.properties[name].required = false;
                shape.properties[name].nullable = true;
            }
            if (Array.isArray(field.yup.checks)) {
                field.yup.checks.forEach(check => {
                    shape.properties[name][check.key] = check.value;
                    if (check.error) {
                        config.errMessages[name][check.key] = check.error;
                    }
                });
            }
        }
    });
    const yup = buildYup(shape, config);
    return yup;
};
//# sourceMappingURL=yupGenerator.js.map