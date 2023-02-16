import { FormControlLabel, FormGroup, Switch, SwitchProps } from "@mui/material";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedSwitch = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering switch');
    const props = useMemo(() => fieldData.props as SwitchProps, [fieldData.props]);

    return (
        <FormGroup key={`field-${fieldData.fieldName}-${index}`}>
            <FormControlLabel control={(
                <Switch
                    disabled={disabled}
                    size={props.size}
                    color={props.color}
                    tabIndex={index}
                    checked={formik.values[fieldData.fieldName]}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    name={fieldData.fieldName}
                    inputProps={{ 'aria-label': fieldData.fieldName }}
                />
            )} label={fieldData.fieldName} />
        </FormGroup>
    );
}