import { InputType } from "@local/consts";
const defaultMap = {
    [InputType.Checkbox]: (props) => ({
        color: "secondary",
        defaultValue: new Array(props.options?.length ?? 0).fill(false),
        options: [],
        row: true,
        ...props,
    }),
    [InputType.Dropzone]: (props) => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.JSON]: (props) => ({
        defaultValue: "",
        ...props,
    }),
    [InputType.LanguageInput]: (props) => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.IntegerInput]: (props) => ({
        ...props,
    }),
    [InputType.Markdown]: (props) => ({
        defaultValue: "",
        ...props,
    }),
    [InputType.Prompt]: (props) => ({
        defaultValue: "",
        ...props,
    }),
    [InputType.Radio]: (props) => ({
        defaultValue: (Array.isArray(props.options) && props.options.length > 0) ? props.options[0].value : "",
        ...props,
    }),
    [InputType.Selector]: (props) => ({
        options: [],
        getOptionLabel: (option) => option,
        ...props,
    }),
    [InputType.Slider]: (props) => {
        let { defaultValue, min, max, step, ...otherProps } = props;
        const isNumeric = (n) => !isNaN(parseFloat(n)) && isFinite(n);
        const nearest = (value, min, max, steps) => {
            let zerone = Math.round((value - min) * steps / (max - min)) / steps;
            zerone = Math.min(Math.max(zerone, 0), 1);
            return zerone * (max - min) + min;
        };
        if (!isNumeric(min))
            min = 0;
        if (!isNumeric(max))
            max = 100;
        if (!isNumeric(step))
            step = (max - min) / 20;
        if (!isNumeric(defaultValue))
            defaultValue = nearest((min + max) / 2, min, max, step || 1);
        return {
            defaultValue,
            min,
            max,
            step,
            ...otherProps,
        };
    },
    [InputType.Switch]: (props) => ({
        defaultValue: false,
        color: "secondary",
        size: "medium",
        ...props,
    }),
    [InputType.TagSelector]: (props) => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.TextField]: (props) => ({
        defaultValue: "",
        ...props,
    }),
};
export const generateDefaultProps = (fields) => {
    if (!fields)
        return [];
    return fields.map(field => {
        const { props, ...otherKeys } = field;
        return {
            props: defaultMap[field.type](props),
            ...otherKeys,
        };
    });
};
export const createDefaultFieldData = ({ fieldName, label, type, yup, }) => {
    if (!type || !defaultMap[type])
        return null;
    return ({
        type: type,
        props: defaultMap[type]({}),
        fieldName: fieldName ?? "",
        label: label ?? "",
        yup: yup ?? ({
            checks: [],
        }),
    });
};
//# sourceMappingURL=defaultGenerator.js.map