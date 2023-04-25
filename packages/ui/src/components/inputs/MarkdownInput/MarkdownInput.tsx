import { useField } from "formik";
import { MarkdownInputBase } from "../MarkdownInputBase/MarkdownInputBase";
import { MarkdownInputProps } from "../types";

export const MarkdownInput = ({
    name,
    ...props
}: MarkdownInputProps) => {
    const [field, meta] = useField(name);

    return (
        <MarkdownInputBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={field.onChange}
        />
    );
};
