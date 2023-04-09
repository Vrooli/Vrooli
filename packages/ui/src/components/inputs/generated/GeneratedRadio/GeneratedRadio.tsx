import { FormControl, FormControlLabel, FormLabel, Radio, RadioGroup } from "@mui/material";
import { useField } from "formik";
import { RadioProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedRadio = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering radio');
    const [field] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props as RadioProps, [fieldData.props]);

    return (
        <FormControl
            component="fieldset"
            disabled={disabled}
            key={`field-${fieldData.fieldName}-${index}`}
            sx={{ paddingLeft: 1 }}
        >
            <FormLabel component="legend">{fieldData.label}</FormLabel>
            <RadioGroup
                aria-label={fieldData.fieldName}
                row={props.row}
                id={fieldData.fieldName}
                name={fieldData.fieldName}
                value={field.value ?? props.defaultValue}
                defaultValue={props.defaultValue}
                onBlur={field.onBlur}
                onChange={field.onChange}
                tabIndex={index}
            >
                {
                    props.options.map((option, index) => (
                        <FormControlLabel
                            key={index}
                            value={option.value}
                            control={<Radio />}
                            label={option.label}
                        />
                    ))
                }
            </RadioGroup>
        </FormControl>

    );
}