import React from 'react'
import {
    Checkbox,
    FormControl,
    FormControlLabel,
    FormGroup,
    FormHelperText,
    FormLabel,
    Radio,
    RadioGroup,
    Slider,
    Stack,
    Switch,
    TextField,
    Typography
} from '@mui/material'
import { Dropzone, JsonInput, LanguageInput, MarkdownInput, QuantityBox, Selector, TagSelector } from 'components'
import { DropzoneProps, QuantityBoxProps, SelectorProps } from 'components/inputs/types'
import { CheckboxProps, FieldData, JSONProps, MarkdownProps, RadioProps, SliderProps, SwitchProps, TextFieldProps } from 'forms/types'
import { TagShape, updateArray } from 'utils'
import { InputType, isObject } from '@local/shared'
import { Session } from 'types'

/**
 * Function signature shared between all input components
 */
export interface InputGeneratorProps {
    data: FieldData,
    disabled?: boolean,
    formik: any,
    index?: number,
    onUpload: (fieldName: string, files: string[]) => void,
    session: Session;
    zIndex: number
}

/**
 * Converts JSON into a Checkbox group component.
 * Uses formik for validation and onChange.
 */
export const toCheckbox = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as CheckboxProps;
    const hasError: boolean = formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName]);
    const errorText: string | null = hasError ? formik.errors[data.fieldName] : null;
    return (
        <FormControl
            key={`field-${data.fieldName}-${index}`}
            disabled={disabled}
            required={data.yup?.required}
            error={hasError}
            component="fieldset"
            variant="standard"
            sx={{ m: 3 }}
        >
            <FormLabel component="legend">{data.label}</FormLabel>
            <FormGroup row={props.row ?? false}>
                {
                    props.options.map((option, index) => (
                        <FormControlLabel
                            control={
                                <Checkbox
                                    checked={(Array.isArray(formik.values[data.fieldName]) && formik.values[data.fieldName].length > index) ? formik.values[data.fieldName][index] === props.options[index] : false}
                                    onChange={(event) => { formik.setFieldValue(data.fieldName, updateArray(formik.values[data.fieldName], index, !props.options[index])) }}
                                    name={`${data.fieldName}-${index}`}
                                    id={`${data.fieldName}-${index}`}
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

/**
 * Converts JSON into a Dropzone component.
 */
export const toDropzone = ({
    data,
    disabled,
    index,
    onUpload,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as DropzoneProps;
    return (
        <Stack direction="column" key={`field-${data.fieldName}-${index}`} spacing={1}>
            <Typography variant="h5" textAlign="center">{data.label}</Typography>
            <Dropzone
                acceptedFileTypes={props.acceptedFileTypes}
                disabled={disabled}
                dropzoneText={props.dropzoneText}
                uploadText={props.uploadText}
                cancelText={props.cancelText}
                maxFiles={props.maxFiles}
                showThumbs={props.showThumbs}
                onUpload={(files) => { onUpload(data.fieldName, files) }}
            />
        </Stack>
    );
}

/**
 * Converts JSON into a JSON input component. This component 
 * allows you to view the JSON format, insert JSON that matches, 
 * and highlights the incorrect parts of the input.
 * Uses formik for validation and onChange.
 */
export const toJSON = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as JSONProps;
    return (
        <JsonInput
            id={data.fieldName}
            disabled={disabled}
            format={props.format}
            variables={props.variables}
            placeholder={props.placeholder ?? data.label}
            value={formik.values[data.fieldName]}
            minRows={props.minRows}
            onChange={(newText: string) => formik.setFieldValue(data.fieldName, newText)}
            error={formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName])}
            helperText={formik.touched[data.fieldName] && formik.errors[data.fieldName]}
        />
    )
}

/**
 * Converts JSON into a LanguageInput component.
 * Uses formik for validation and onChange.
 */
export const toLanguageInput = ({
    data,
    disabled,
    formik,
    index,
    session,
    zIndex,
}: InputGeneratorProps): React.ReactElement => {
    let languages: string[] = [];
    if (isObject(formik.values) && Array.isArray(formik.values[data.fieldName]) && data.fieldName in formik.values) {
        languages = formik.values[data.fieldName] as string[];
    }
    const addLanguage = (lang: string) => {
        formik.setFieldValue(data.fieldName, [...languages, lang]);
    };
    const deleteLanguage = (lang: string) => {
        const newLanguages = [...languages.filter(l => l !== lang)]
        formik.setFieldValue(data.fieldName, newLanguages);
    }
    return (
        <LanguageInput
            currentLanguage={languages.length > 0 ? languages[0] : ''} //TOOD
            disabled={disabled}
            handleAdd={addLanguage}
            handleDelete={deleteLanguage}
            handleCurrent={() => {}} //TODO
            selectedLanguages={languages}
            session={session}
            zIndex={zIndex}
        />
    )
}

/**
 * Converts JSON into a markdown input component.
 * Uses formik for validation and onChange.
 */
export const toMarkdown = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as MarkdownProps;
    return (
        <MarkdownInput
            id={data.fieldName}
            disabled={disabled}
            placeholder={props.placeholder ?? data.label}
            value={formik.values[data.fieldName]}
            minRows={props.minRows}
            onChange={(newText: string) => formik.setFieldValue(data.fieldName, newText)}
            error={formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName])}
            helperText={formik.touched[data.fieldName] && formik.errors[data.fieldName]}
        />
    )
}

/**
 * Converts JSON into a Radio component.
 * Uses formik for validation and onChange.
 */
export const toRadio = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as RadioProps;
    return (
        <FormControl component="fieldset" disabled={disabled} key={`field-${data.fieldName}-${index}`}>
            <FormLabel component="legend">{data.label}</FormLabel>
            <RadioGroup
                aria-label={data.fieldName}
                row={props.row}
                id={data.fieldName}
                name={data.fieldName}
                value={formik.values[data.fieldName] ?? props.defaultValue}
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

/**
 * Converts JSON into a Selector component.
 * Uses formik for validation and onChange.
 */
export const toSelector = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as SelectorProps;
    return (
        <Selector
            key={`field-${data.fieldName}-${index}`}
            disabled={disabled}
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
            tabIndex={index}
        />
    );
}

/**
 * Converts JSON into a Slider component.
 * Uses formik for validation and onChange.
 */
export const toSlider = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as SliderProps;
    return (
        <Slider
            aria-label={data.fieldName}
            disabled={disabled}
            key={`field-${data.fieldName}-${index}`}
            min={props.min}
            max={props.max}
            step={props.step}
            valueLabelDisplay={props.valueLabelDisplay}
            value={formik.values[data.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
            tabIndex={index}
        />
    );
}

/**
 * Converts JSON into a Switch component.
 * Uses formik for validation and onChange.
 */
export const toSwitch = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as SwitchProps;
    return (
        <FormGroup key={`field-${data.fieldName}-${index}`}>
            <FormControlLabel control={(
                <Switch
                    disabled={disabled}
                    size={props.size}
                    color={props.color}
                    tabIndex={index}
                    checked={formik.values[data.fieldName]}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    name={data.fieldName}
                    inputProps={{ 'aria-label': data.fieldName }}
                />
            )} label={data.fieldName} />
        </FormGroup>
    );
}

/**
 * Converts JSON into a TagSelector component.
 * Uses formik for validation and onChange.
 */
export const toTagSelector = ({
    data,
    disabled,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const tags = formik.values[data.fieldName] as TagShape[];
    const addTag = (tag: TagShape) => {
        formik.setFieldValue(data.fieldName, [...tags, tag]);
    };
    const removeTag = (tag: TagShape) => {
        formik.setFieldValue(data.fieldName, tags.filter((t) => t.tag !== tag.tag));
    };
    const clearTags = () => {
        formik.setFieldValue(data.fieldName, []);
    };
    return (
        <TagSelector
            disabled={disabled}
            session={{}} //TODO
            tags={tags}
            onTagAdd={addTag}
            onTagRemove={removeTag}
            onTagsClear={clearTags}
        />
    );
}

/**
 * Converts JSON into a TextField component.
 * Uses formik for validation and onChange.
 * If this component is first in the list, it will be focused.
 */
export const toTextField = ({
    data,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as TextFieldProps;
    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    const hasDescription = typeof data.description === 'string' && data.description.trim().length > 0;
    return (
        <TextField
            key={`field-${data.fieldName}-${index}`}
            autoComplete={props.autoComplete}
            autoFocus={index === 0}
            fullWidth
            id={data.fieldName}
            InputLabelProps={{ shrink: true }}
            name={data.fieldName}
            placeholder={hasDescription ? data.description : `${data.label}...`}
            required={data.yup?.required}
            value={formik.values[data.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
            tabIndex={index}
            error={formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName])}
            helperText={formik.touched[data.fieldName] && formik.errors[data.fieldName]}
            {...multiLineProps}
        />
    );
}

/**
 * Converts JSON into a QuantityBox (number input) component.
 * Uses formik for validation and onChange.
 * If this component is first in the list, it will be focused.
 */
export const toQuantityBox = ({
    data,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as QuantityBoxProps;
    return (
        <QuantityBox
            autoFocus={index === 0}
            key={`field-${data.fieldName}-${index}`}
            tabIndex={index}
            id={data.fieldName}
            label={data.label}
            min={props.min ?? 0}
            tooltip={props.tooltip ?? ''}
            value={formik.values[data.fieldName]}
            onBlur={formik.handleBlur}
            handleChange={(value: number) => formik.setFieldValue(data.fieldName, value)}
            error={formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName])}
            helperText={formik.touched[data.fieldName] && formik.errors[data.fieldName]}
        />
    );
}

/**
 * Maps a data input type string to its corresponding component generator function
 */
const typeMap: { [key in InputType]: (props: InputGeneratorProps) => any } = {
    [InputType.Checkbox]: toCheckbox,
    [InputType.Dropzone]: toDropzone,
    [InputType.JSON]: toJSON,
    [InputType.LanguageInput]: toLanguageInput,
    [InputType.Markdown]: toMarkdown,
    [InputType.Radio]: toRadio,
    [InputType.Selector]: toSelector,
    [InputType.Slider]: toSlider,
    [InputType.Switch]: toSwitch,
    [InputType.TagSelector]: toTagSelector,
    [InputType.TextField]: toTextField,
    [InputType.QuantityBox]: toQuantityBox,
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
 */
export const generateInputComponent = ({
    data,
    disabled,
    formik,
    index,
    onUpload,
    session,
    zIndex,
}: InputGeneratorProps): React.ReactElement | null => {
    return typeMap[data.type]({ data, disabled, formik, index, onUpload, session, zIndex });
}