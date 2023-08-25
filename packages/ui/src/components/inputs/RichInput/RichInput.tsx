import { useField } from "formik";
import { RichInputBase } from "../RichInputBase/RichInputBase";
import { RichInputProps } from "../types";

export const RichInput = ({
    name,
    ...props
}: RichInputProps) => {
    const [field, meta, helpers] = useField(name);

    const handleChange = (value) => {
        helpers.setValue(value);
    };

    return (
        <RichInputBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
};
