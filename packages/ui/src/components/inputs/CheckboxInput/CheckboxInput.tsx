import { Checkbox, FormControlLabel } from "@mui/material";
import { type CheckboxInputProps } from "../types.js";

export function CheckboxInput({
    disabled,
    label,
    field,
    ...props
}: CheckboxInputProps) {
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
}
