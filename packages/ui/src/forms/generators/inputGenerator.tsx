import React from 'react'
import {
    Checkbox,
    FormControl,
    FormControlLabel,
    FormGroup,
    FormLabel,
    Radio,
    RadioGroup,
    Slider,
    Stack,
    Switch,
    TextField,
    Typography
} from '@mui/material'
import { Dropzone, Selector } from 'components'
import { DropzoneProps, SelectorProps } from 'components/inputs/types'
import { CheckboxProps, FieldData, InputType, RadioProps, SliderProps, SwitchProps, TextFieldProps } from 'forms/types'

/**
 * Converts JSON into a Checkbox component.
 * Uses formik for validation and onChange.
 */
export const toCheckbox = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as CheckboxProps;
    return (
        <FormControlLabel
            control={
                <Checkbox
                    id={data.fieldName}
                    name={data.fieldName}
                    color={props.color}
                    checked={formik.values[data.fieldName]}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                />
            }
            label={data.label}
        />
    );
}

/**
 * Converts JSON into a Dropzone component.
 */
export const toDropzone = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as DropzoneProps;
    return (
        <Stack direction="column" spacing={1}>
            <Typography variant="h5" textAlign="center">{data.label}</Typography>
            <Dropzone
                acceptedFileTypes={props.acceptedFileTypes}
                dropzoneText={props.dropzoneText}
                uploadText={props.uploadText}
                cancelText={props.cancelText}
                maxFiles={props.maxFiles}
                showThumbs={props.showThumbs}
                onUpload={() => { }} //TODO
            />
        </Stack>
    );
}

/**
 * Converts JSON into a Radio component.
 * Uses formik for validation and onChange.
 */
export const toRadio = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as RadioProps;
    return (
        <FormControl component="fieldset">
            <FormLabel component="legend">{data.label}</FormLabel>
            <RadioGroup
                aria-label={data.fieldName}
                name="radio-buttons-group"
                row={props.row}
                value={formik.values[data.fieldName]}
                onBlur={formik.handleBlur}
                onChange={formik.handleChange}
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

/**
 * Converts JSON into a Selector component.
 * Uses formik for validation and onChange.
 */
export const toSelector = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as SelectorProps;
    return (
        <Selector
            options={props.options}
            getOptionLabel={props.getOptionLabel}
            selected={formik.values[data.fieldName]}
            onBlur={formik.handleBlur}
            handleChange={formik.handleChange}
            fullWidth
            multiple={props.multiple}
            inputAriaLabel={`select-input-${data.fieldName}`}
            noneOption={props.noneOption}
            label={data.label}
            color={props.color}
        />
    );
}

/**
 * Converts JSON into a Slider component.
 * Uses formik for validation and onChange.
 */
export const toSlider = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as SliderProps;
    return (
        <Slider
            aria-label={data.fieldName}
            min={props.min}
            max={props.max}
            step={props.step}
            valueLabelDisplay={props.valueLabelDisplay}
            value={formik.values[data.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
        />
    );
}

/**
 * Converts JSON into a Switch component.
 * Uses formik for validation and onChange.
 */
export const toSwitch = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as SwitchProps;
    return (
        <FormGroup>
            <FormControlLabel control={(
                <Switch
                    size={props.size}
                    color={props.color}
                />
            )} label={data.fieldName} />
        </FormGroup>
    );
}

/**
 * Converts JSON into a TextField component.
 * Uses formik for validation and onChange.
 * If this component is first in the list, it will be focused.
 */
export const toTextField = (
    data: FieldData,
    formik: any,
    index: number
): React.ReactElement => {
    const props = data.props as TextFieldProps;
    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    return (
        <TextField
            fullWidth
            autoFocus={index === 0}
            id={data.fieldName}
            name={data.fieldName}
            required={props.required}
            autoComplete={props.autoComplete}
            label={data.label}
            value={formik.values[data.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
            error={formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName])}
            helperText={formik.touched[data.fieldName] && formik.errors[data.fieldName]}
            {...multiLineProps}
        />
    );
}

/**
 * Maps a data input type string to its corresponding component generator function
 */
 const typeMap = {
    [InputType.Checkbox]: toCheckbox,
    [InputType.Dropzone]: toDropzone,
    [InputType.Radio]: toRadio,
    [InputType.Selector]: toSelector,
    [InputType.Slider]: toSlider,
    [InputType.Switch]: toSwitch,
    [InputType.TextField]: toTextField,
}

/**
 * Converts input data into a component. Current options are:
 * - Checkbox
 * - Dropzone
 * - Radio
 * - Selector
 * - Slider
 * - Switch
 * - TextField
 * @param data The data to convert
 * @param formik The formik object
 * @param index The index of the component (mainly for autoFocus)
 */
export const generateInputComponent = (data: FieldData, formik: any, index: number): React.ReactElement | null => {
    return typeMap[data.type](data, formik, index);
}