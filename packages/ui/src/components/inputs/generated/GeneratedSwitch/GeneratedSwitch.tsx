import { FormControlLabel, FormGroup, Switch, SwitchProps } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedSwitch = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering switch');
    const [field] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props as SwitchProps, [fieldData.props]);

    return (
        <FormGroup key={`field-${fieldData.fieldName}-${index}`}>
            <FormControlLabel tabIndex={index} control={(
                <Switch
                    autoFocus={index === 0}
                    disabled={disabled}
                    size={props.size}
                    color={props.color}
                    tabIndex={index}
                    checked={field.value ?? props.defaultValue}
                    onBlur={field.onBlur}
                    onChange={field.onChange}
                    name={fieldData.fieldName}
                    inputProps={{ 'aria-label': fieldData.fieldName }}
                />
            )} label={fieldData.fieldName} />
        </FormGroup>
    );
}