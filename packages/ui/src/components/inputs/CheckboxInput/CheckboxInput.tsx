import { Checkbox, FormControlLabel } from "@mui/material";
import { CheckboxInputProps } from "../types";

export const CheckboxInput = ({
    disabled,
    label,
    field,
    ...props
}: CheckboxInputProps) => {
    console.log("rendering checkbox input", name, field.value);

    return (
        <FormControlLabel
            control={
                <Checkbox
                    {...props}
                    {...field}
                    checked={field.value}
                />
            }
            label={label}
            disabled={disabled}
        />
    );
};
