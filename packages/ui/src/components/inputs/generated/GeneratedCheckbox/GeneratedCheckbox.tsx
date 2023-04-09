import { Checkbox, FormControl, FormControlLabel, FormGroup, FormHelperText, FormLabel } from "@mui/material";
import { useField } from "formik";
import { CheckboxProps } from "forms/types";
import { useMemo } from "react";
import { updateArray } from "utils/shape/general";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedCheckbox = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    const [field, meta] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props as CheckboxProps, [fieldData.props]);

    console.log('rendering checkbox');

    return (
        <FormControl
            key={`field-${fieldData.fieldName}-${index}`}
            disabled={disabled}
            required={fieldData.yup?.required}
            error={meta.touched && !!meta.error}
            name={fieldData.fieldName}
            component="fieldset"
            variant="standard"
            sx={{ m: 3 }}
        >
            <FormLabel component="legend">{fieldData.label}</FormLabel>
            <FormGroup row={props.row ?? false}>
                {
                    props.options.map((option, index) => (
                        <FormControlLabel
                            control={
                                <Checkbox
                                    checked={(Array.isArray(field.value?.[fieldData.fieldName]) && field.value[fieldData.fieldName].length > index) ? field.value[fieldData.fieldName][index] === props.options[index] : false}
                                    onChange={(event) => { field.onChange(updateArray(field.value[fieldData.fieldName], index, !props.options[index])) }}
                                    name={`${fieldData.fieldName}-${index}`}
                                    id={`${fieldData.fieldName}-${index}`}
                                    value={props.options[index]}
                                />
                            }
                            label={option.label}
                        />
                    ))
                }
            </FormGroup>
            {meta.touched && !!meta.error && <FormHelperText>{meta.error}</FormHelperText>}
        </FormControl>
    )
}