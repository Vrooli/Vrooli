import { Checkbox, FormControl, FormControlLabel, FormGroup, FormHelperText, FormLabel } from "@mui/material";
import { CheckboxProps } from "forms/types";
import { useMemo } from "react";
import { updateArray } from "utils";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedCheckbox = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering checkbox');
    const props = useMemo(() => fieldData.props as CheckboxProps, [fieldData.props]);
    const hasError: boolean = formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName]);
    const errorText: string | null = hasError ? formik.errors[fieldData.fieldName] : null;

    return (
        <FormControl
            key={`field-${fieldData.fieldName}-${index}`}
            disabled={disabled}
            required={fieldData.yup?.required}
            error={hasError}
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
                                    checked={(Array.isArray(formik.values[fieldData.fieldName]) && formik.values[fieldData.fieldName].length > index) ? formik.values[fieldData.fieldName][index] === props.options[index] : false}
                                    onChange={(event) => { formik.setFieldValue(fieldData.fieldName, updateArray(formik.values[fieldData.fieldName], index, !props.options[index])) }}
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
            {errorText && <FormHelperText>{errorText}</FormHelperText>}
        </FormControl>
    )
}