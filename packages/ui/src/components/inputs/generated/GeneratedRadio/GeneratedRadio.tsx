import { FormControl, FormControlLabel, FormLabel, Radio, RadioGroup } from "@mui/material";
import { RadioProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedRadio = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering radio');
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
                value={formik.values[fieldData.fieldName] ?? props.defaultValue}
                defaultValue={props.defaultValue}
                onBlur={formik.handleBlur}
                onChange={formik.handleChange}
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