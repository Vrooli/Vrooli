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
import { Dropzone, JsonFormatInput, LanguageInput, MarkdownInput, QuantityBox, Selector, TagSelector } from 'components'
import { DropzoneProps, QuantityBoxProps, SelectorProps, TagSelectorTag } from 'components/inputs/types'
import { CheckboxProps, FieldData, JSONProps, MarkdownProps, RadioProps, SliderProps, SwitchProps, TextFieldProps } from 'forms/types'
import { updateArray } from 'utils'
import _ from 'lodash'
import { InputType } from '@local/shared'

/**
 * Function signature shared between all input components
 */
export interface InputGeneratorProps {
    data: FieldData,
    formik: any,
    index: number,
    onUpload: (fieldName: string, files: string[]) => void,
}

/**
 * Converts JSON into a Checkbox group component.
 * Uses formik for validation and onChange.
 */
export const toCheckbox = ({
    data,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as CheckboxProps;
    const hasError: boolean = formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName]);
    const errorText: string | null = hasError ? formik.errors[data.fieldName] : null;
    return (
        <FormControl
            key={`field-${data.fieldName}-${index}`}
            required={data.yup?.required}
            error={hasError}
            component="fieldset"
            sx={{ m: 3 }}
            variant="standard"
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

// /**
//  * Converts JSON into a Checkbox component.
//  * Uses formik for validation and onChange.
//  */
// export const toCheckbox = ({
//     data,
//     formik,
//     index,
// }: InputGeneratorProps): React.ReactElement => {
//     const props = data.props as CheckboxProps;
//     return (
//         <FormControlLabel
//             key={`field-${data.fieldName}-${index}`}
//             control={
//                 <Checkbox
//                     id={data.fieldName}
//                     name={data.fieldName}
//                     color={props.color}
//                     checked={formik.values[data.fieldName]}
//                     onBlur={formik.handleBlur}
//                     onChange={formik.handleChange}
//                     tabIndex={index}
//                 />
//             }
//             label={data.label}
//         />
//     );
// }

/**
 * Converts JSON into a Dropzone component.
 */
export const toDropzone = ({
    data,
    index,
    onUpload,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as DropzoneProps;
    return (
        <Stack direction="column" key={`field-${data.fieldName}-${index}`} spacing={1}>
            <Typography variant="h5" textAlign="center">{data.label}</Typography>
            <Dropzone
                acceptedFileTypes={props.acceptedFileTypes}
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
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as JSONProps;
    return (
        <JsonFormatInput
            id={data.fieldName}
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
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    let languages: string[] = [];
    if (_.isObject(formik.values) && Array.isArray(formik.values[data.fieldName]) && data.fieldName in formik.values) {
        languages = formik.values[data.fieldName] as string[];
    }
    const addLanguage = (lang: string) => {
        formik.setFieldValue(data.fieldName, [...languages, lang]);
    };
    const changeLanguage = (oldLang: string, newLang: string) => {
        const newLanguages = [...languages];
        const index = newLanguages.findIndex(l => l === oldLang);
        if (index >= 0) {
            newLanguages[index] = newLang;
            formik.setFieldValue(data.fieldName, newLanguages);
        }
    }
    const deleteLanguage = (lang: string) => {
        const newLanguages = [...languages.filter(l => l !== lang)]
        formik.setFieldValue(data.fieldName, newLanguages);
    }
    return (
        <LanguageInput
            handleAdd={addLanguage}
            handleChange={changeLanguage}
            handleDelete={deleteLanguage}
            handleSelect={() => {}}
            languages={languages}
            session={{}} //TODO
        />
    )
}

/**
 * Converts JSON into a markdown input component.
 * Uses formik for validation and onChange.
 */
export const toMarkdown = ({
    data,
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as MarkdownProps;
    return (
        <MarkdownInput
            id={data.fieldName}
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
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as RadioProps;
    return (
        <FormControl component="fieldset" key={`field-${data.fieldName}-${index}`}>
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
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as SelectorProps;
    return (
        <Selector
            key={`field-${data.fieldName}-${index}`}
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
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as SliderProps;
    return (
        <Slider
            aria-label={data.fieldName}
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
    index,
}: InputGeneratorProps): React.ReactElement => {
    const props = data.props as SwitchProps;
    return (
        <FormGroup key={`field-${data.fieldName}-${index}`}>
            <FormControlLabel control={(
                <Switch
                    size={props.size}
                    color={props.color}
                    tabIndex={index}
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
    formik,
    index,
}: InputGeneratorProps): React.ReactElement => {
    const tags = formik.values[data.fieldName] as TagSelectorTag[];
    const addTag = (tag: TagSelectorTag) => {
        formik.setFieldValue(data.fieldName, [...tags, tag]);
    };
    const removeTag = (tag: TagSelectorTag) => {
        formik.setFieldValue(data.fieldName, tags.filter((t) => t.tag !== tag.tag));
    };
    const clearTags = () => {
        formik.setFieldValue(data.fieldName, []);
    };
    return (
        <TagSelector
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
    return (
        <TextField
            key={`field-${data.fieldName}-${index}`}
            fullWidth
            autoFocus={index === 0}
            tabIndex={index}
            id={data.fieldName}
            name={data.fieldName}
            required={data.yup?.required}
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
 * @param data The data to convert
 * @param formik The formik object
 * @param index The index of the component (for autoFocus & tabIndex)
 * @param onUpload Callback for uploading files
 */
export const generateInputComponent = ({
    data,
    formik,
    index,
    onUpload,
}: InputGeneratorProps): React.ReactElement | null => {
    return typeMap[data.type]({ data, formik, index, onUpload });
}