import React from 'react'
import {
    Box,
    Checkbox,
    FormControl,
    FormControlLabel,
    FormGroup,
    FormHelperText,
    FormLabel,
    IconButton,
    Radio,
    RadioGroup,
    Slider,
    Stack,
    Switch,
    TextField,
    Tooltip,
    Typography
} from '@mui/material'
import { Dropzone, HelpButton, JsonInput, LanguageInput, MarkdownInput, QuantityBox, Selector, TagSelector } from 'components'
import { DropzoneProps, QuantityBoxProps, SelectorProps } from 'components/inputs/types'
import { CheckboxProps, FieldData, JSONProps, MarkdownProps, RadioProps, SliderProps, SwitchProps, TextFieldProps } from 'forms/types'
import { TagShape, updateArray } from 'utils'
import { InputType } from '@shared/consts';
import { isObject } from '@shared/utils';
import { Session } from 'types'
import { CopyIcon } from '@shared/icons'

/**
 * Function signature shared between all input components
 */
export interface InputGeneratorProps {
    disabled?: boolean;
    fieldData: FieldData;
    formik: any;
    index?: number;
    onUpload: (fieldName: string, files: string[]) => void;
    session: Session;
    zIndex: number;
}

export interface GenerateInputWithLabelProps extends InputGeneratorProps {
    copyInput?: (fieldName: string) => void;
    textPrimary: string;
}

/**
 * Converts JSON into a Checkbox group component.
 * Uses formik for validation and onChange.
 */
export const toCheckbox = ({
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as CheckboxProps;
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

/**
 * Converts JSON into a Dropzone component.
 */
export const toDropzone = ({
    disabled,
    fieldData,
    index,
    onUpload,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as DropzoneProps;
    return (
        <Stack direction="column" key={`field-${fieldData.fieldName}-${index}`} spacing={1}>
            <Typography variant="h5" textAlign="center">{fieldData.label}</Typography>
            <Dropzone
                acceptedFileTypes={props.acceptedFileTypes}
                disabled={disabled}
                dropzoneText={props.dropzoneText}
                uploadText={props.uploadText}
                cancelText={props.cancelText}
                maxFiles={props.maxFiles}
                showThumbs={props.showThumbs}
                onUpload={(files) => { onUpload(fieldData.fieldName, files) }}
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
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as JSONProps;
    return (
        <JsonInput
            id={fieldData.fieldName}
            disabled={disabled}
            format={props.format}
            variables={props.variables}
            placeholder={props.placeholder ?? fieldData.label}
            value={formik.values[fieldData.fieldName]}
            minRows={props.minRows}
            onChange={(newText: string) => formik.setFieldValue(fieldData.fieldName, newText)}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
        />
    )
}

/**
 * Converts JSON into a LanguageInput component.
 * Uses formik for validation and onChange.
 */
export const toLanguageInput = ({
    disabled,
    fieldData,
    formik,
    index,
    session,
    zIndex,
}: InputGeneratorProps): React.ReactElement => {
    let languages: string[] = [];
    if (isObject(formik.values) && Array.isArray(formik.values[fieldData.fieldName]) && fieldData.fieldName in formik.values) {
        languages = formik.values[fieldData.fieldName] as string[];
    }
    const addLanguage = (lang: string) => {
        formik.setFieldValue(fieldData.fieldName, [...languages, lang]);
    };
    const deleteLanguage = (lang: string) => {
        const newLanguages = [...languages.filter(l => l !== lang)]
        formik.setFieldValue(fieldData.fieldName, newLanguages);
    }
    return (
        <LanguageInput
            currentLanguage={languages.length > 0 ? languages[0] : ''} //TOOD
            disabled={disabled}
            handleAdd={addLanguage}
            handleDelete={deleteLanguage}
            handleCurrent={() => { }} //TODO
            session={session}
            translations={languages.map(language => ({ language }))}
            zIndex={zIndex}
        />
    )
}

/**
 * Converts JSON into a markdown input component.
 * Uses formik for validation and onChange.
 */
export const toMarkdown = ({
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as MarkdownProps;
    return (
        <MarkdownInput
            id={fieldData.fieldName}
            disabled={disabled}
            placeholder={props.placeholder ?? fieldData.label}
            value={formik.values[fieldData.fieldName]}
            minRows={props.minRows}
            onChange={(newText: string) => formik.setFieldValue(fieldData.fieldName, newText)}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
        />
    )
}

/**
 * Converts JSON into a Radio component.
 * Uses formik for validation and onChange.
 */
export const toRadio = ({
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as RadioProps;
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

/**
 * Converts JSON into a Selector component.
 * Uses formik for validation and onChange.
 */
export const toSelector = ({
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as SelectorProps;
    return (
        <Selector
            key={`field-${fieldData.fieldName}-${index}`}
            disabled={disabled}
            options={props.options}
            getOptionLabel={props.getOptionLabel}
            selected={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            handleChange={formik.handleChange}
            fullWidth
            multiple={props.multiple}
            inputAriaLabel={`select-input-${fieldData.fieldName}`}
            noneOption={props.noneOption}
            label={fieldData.label}
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
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as SliderProps;
    return (
        <Slider
            aria-label={fieldData.fieldName}
            disabled={disabled}
            key={`field-${fieldData.fieldName}-${index}`}
            min={props.min}
            max={props.max}
            step={props.step}
            valueLabelDisplay={props.valueLabelDisplay}
            value={formik.values[fieldData.fieldName]}
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
    disabled,
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as SwitchProps;
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

/**
 * Converts JSON into a TagSelector component.
 * Uses formik for validation and onChange.
 */
export const toTagSelector = ({
    disabled,
    fieldData,
    formik,
    index,
    session,
}: InputGeneratorProps): React.ReactElement => {
    const tags = formik.values[fieldData.fieldName] as TagShape[];
    const handleTagsUpdate = (updatedList: TagShape[]) => { formik.setFieldValue(fieldData.fieldName, updatedList) };
    return (
        <TagSelector
            disabled={disabled}
            handleTagsUpdate={handleTagsUpdate}
            session={session}
            tags={tags}
        />
    );
}

/**
 * Converts JSON into a TextField component.
 * Uses formik for validation and onChange.
 * If this component is first in the list, it will be focused.
 */
export const toTextField = ({
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as TextFieldProps;
    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    const hasDescription = typeof fieldData.description === 'string' && fieldData.description.trim().length > 0;
    return (
        <TextField
            key={`field-${fieldData.fieldName}-${index}`}
            autoComplete={props.autoComplete}
            autoFocus={index === 0}
            fullWidth
            id={fieldData.fieldName}
            InputLabelProps={{ shrink: true }}
            name={fieldData.fieldName}
            placeholder={hasDescription ? fieldData.description as string : `${fieldData.label}...`}
            required={fieldData.yup?.required}
            value={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
            tabIndex={index}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
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
    fieldData,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = fieldData.props as QuantityBoxProps;
    return (
        <QuantityBox
            autoFocus={index === 0}
            key={`field-${fieldData.fieldName}-${index}`}
            tabIndex={index}
            id={fieldData.fieldName}
            label={fieldData.label}
            min={props.min ?? 0}
            tooltip={props.tooltip ?? ''}
            value={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            handleChange={(value: number) => formik.setFieldValue(fieldData.fieldName, value)}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
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
    disabled,
    fieldData,
    formik,
    index,
    onUpload,
    session,
    zIndex,
}: InputGeneratorProps): React.ReactElement | null =>
    typeMap[fieldData.type]({ disabled, fieldData, formik, index, onUpload, session, zIndex });

/**
 * Generates an input component, with a copy button, label, and help button
 */
export const generateInputWithLabel = ({
    copyInput,
    disabled,
    fieldData,
    formik,
    index,
    onUpload,
    session,
    textPrimary,
    zIndex,
}: GenerateInputWithLabelProps): React.ReactElement | null => {
    return (<Box key={index} sx={{
        paddingTop: 1,
        paddingBottom: 1,
        borderRadius: 1,
    }}>
        {/* Label, help button, and copy iput icon */}
        <Stack direction="row" spacing={0} sx={{ alignItems: 'center' }}>
            <Tooltip title="Copy to clipboard">
                <IconButton onClick={() => copyInput && copyInput(fieldData.fieldName)}>
                    <CopyIcon fill={textPrimary} />
                </IconButton>
            </Tooltip>
            <Typography variant="h6" sx={{ color: textPrimary }}>{fieldData.label ?? (index && `Input ${index + 1}`) ?? 'Input'}</Typography>
            {fieldData.helpText && <HelpButton markdown={fieldData.helpText} />}
        </Stack>
        {
            generateInputComponent({
                fieldData,
                disabled: false,
                formik: formik,
                index,
                session,
                onUpload: () => { },
                zIndex,
            })
        }
    </Box>)
}