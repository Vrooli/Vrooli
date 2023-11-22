import { Checkbox, FormControlLabel } from "@mui/material";
import { CheckboxInputProps } from "../types";

export const CheckboxInput = ({
    disabled,
    label,
    field,
    ...props
}: CheckboxInputProps) => {
    return (
        <FormControlLabel
            control={
                <Checkbox
                    {...props}
                    {...field}
                    checked={field.value}
                    onChange={(event) => {
                        // Explicitly set the value to true or false
                        field.onChange({
                            ...event,
                            target: {
                                ...event.target,
                                value: event.target.checked,
                                name: field.name,
                            },
                        });
                    }}
                />
            }
            label={label}
            disabled={disabled}
        />
    );
};
