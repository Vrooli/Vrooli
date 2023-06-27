import { useField } from "formik";
import { CodeInputBase } from "../CodeInputBase/CodeInputBase";
import { CodeInputBaseProps, CodeInputProps } from "../types";

export const CodeInput = ({
    ...props
}: CodeInputProps) => {
    console.log("codeinput", props.limitTo);
    const [defaultValueField] = useField<CodeInputBaseProps["defaultValue"]>("defaultValue");
    const [formatField] = useField<CodeInputBaseProps["format"]>("format");
    const [variablesField] = useField<CodeInputBaseProps["variables"]>("variables");

    return (
        <CodeInputBase
            {...props}
            defaultValue={defaultValueField.value}
            format={formatField.value}
            variables={variablesField.value}
        />
    );
};
