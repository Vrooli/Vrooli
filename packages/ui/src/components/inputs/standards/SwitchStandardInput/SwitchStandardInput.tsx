
import { SwitchStandardInputProps } from '../types';
import { switchStandardInputForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { FormControlLabel, FormGroup, Switch } from '@mui/material';

/**
 * Input for entering (and viewing format of) Switch data that 
 * must match a certain schema.
 */
export const SwitchStandardInput = ({
    defaultValue,
    isEditing,
    onPropsChange,
}: SwitchStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: defaultValue ?? false,
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onPropsChange({
            ...formik.values,
        });
    }, [formik.values, onPropsChange]);

    return (
        <FormGroup>
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