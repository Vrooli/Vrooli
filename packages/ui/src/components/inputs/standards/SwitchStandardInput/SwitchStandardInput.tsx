/**
 * Input for entering (and viewing format of) Switch data that 
 * must match a certain schema.
 */
import { SwitchStandardInputProps } from '../types';
import { InputType, switchStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { FormControlLabel, FormGroup, Switch } from '@mui/material';

export const SwitchStandardInput = ({
    isEditing,
    schema,
    onChange,
}: SwitchStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? false,
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.Switch,
            props: formik.values,
            fieldName: schema.fieldName,
            label: schema.label,
            yup: schema.yup ?? {
                checks: [],
            }
        });
    }, [formik.values, onChange, schema.fieldName, schema.label, schema.yup]);

    return (
        <FormGroup key={schema.fieldName}>
            <FormControlLabel control={(
                <Switch
                    disabled={!isEditing}
                    size={'medium'}
                    color="secondary"
                    checked={formik.values.defaultValue}
                    onChange={(event) => {
                        formik.setFieldValue('defaultValue', event.target.checked);
                    }}
                />
            )} label="Default checked" />
        </FormGroup>
    );
}