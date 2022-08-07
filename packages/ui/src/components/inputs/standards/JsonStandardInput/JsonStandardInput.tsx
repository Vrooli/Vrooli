/**
 * Input for entering (and viewing format of) JSON data that 
 * must match a certain schema.
 */
import { JsonStandardInputProps } from '../types';
import { jsonStandardInputForm as validationSchema } from '@local/shared';
import { useEffect, useMemo, useState } from 'react';
import { useFormik } from 'formik';
import { Box, IconButton, Stack, TextField, Tooltip, useTheme } from '@mui/material';
import { HelpButton, StatusButton } from 'components/buttons';
import { isEqualJSON, isJson, jsonHelpText, jsonToMarkdown, jsonToString, Status, TERTIARY_COLOR } from 'utils';
import {
    VisibilityOff as PreviewOffIcon,
    Visibility as PreviewOnIcon,
} from '@mui/icons-material';
import Markdown from 'markdown-to-jsx';

export const JsonStandardInput = ({
    defaultValue,
    format,
    isEditing,
    onPropsChange,
    variables,
}: JsonStandardInputProps) => {
    const { palette } = useTheme();

    // Last valid schema format
    const [internalValue, setInternalValue] = useState<string>('{}');

    const formik = useFormik({
        initialValues: {
            format: internalValue,
            defaultValue: defaultValue ?? '',
            variables: variables ?? {},
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    /**
    * Set internal value when format changes
    */
    useEffect(() => {
        if (!isEditing) return;
        // Compare to current internal value
        if (isEqualJSON(formik.values.format, internalValue)) return;
        setInternalValue(jsonToString(format) ?? '');
    }, [format, formik.values.format, internalValue, isEditing]);

    // Check if formik.values.format is valid JSON
    useEffect(() => {
        if (isJson(formik.values.format)) {
            setInternalValue(formik.values.format);
        }
    }, [formik.values.format]);

    // Update format only when it is valid
    useEffect(() => {
        if (internalValue.length > 2) {
            onPropsChange({
                format: JSON.parse(internalValue),
            });
        }
    }, [format, internalValue, onPropsChange]);

    // Update other props separately
    useEffect(() => {
        onPropsChange({
            defaultValue: formik.values.defaultValue,
            variables: formik.values.variables,
        });
    }, [formik.values.defaultValue, formik.values.variables, onPropsChange]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const togglePreview = () => setIsPreviewOn(!isPreviewOn);
    const { previewMarkdown, isValueValid } = useMemo(() => ({
        previewMarkdown: jsonToMarkdown(formik.values.format),
        isValueValid: internalValue.length > 2 && isJson(formik.values.format),
    }), [formik.values.format, internalValue.length]);

    return (
        <Stack direction="column" spacing={0}>
            {/* Bar above TextField, for status and HelpButton */}
            <Box sx={{
                display: 'flex',
                width: '100%',
                padding: '0.5rem',
                borderBottom: '1px solid #e0e0e0',
                background: palette.primary.light,
                color: palette.primary.contrastText,
                borderRadius: '0.5rem 0.5rem 0 0',
            }}>
                <StatusButton
                    status={isValueValid ? Status.Valid : Status.Invalid}
                    messages={isValueValid ? ['JSON is valid'] : ['JSON is empty or could not be parsed']}
                    sx={{
                        marginLeft: 1,
                        marginRight: 'auto',
                    }}
                />
                {/* Toggle preview */}
                <Tooltip title={isPreviewOn ? 'Preview mode' : 'Edit mode'} placement="top" sx={{ marginLeft: 'auto' }}>
                    <IconButton size="small" onClick={togglePreview}>
                        {
                            isPreviewOn ?
                                <PreviewOffIcon sx={{ fill: palette.primary.contrastText }} /> :
                                <PreviewOnIcon sx={{ fill: palette.primary.contrastText }} />
                        }
                    </IconButton>
                </Tooltip>
                <HelpButton
                    markdown={jsonHelpText}
                    sxRoot={{ marginRight: 1 }}
                    sx={{ color: TERTIARY_COLOR }}
                />
            </Box>
            {/* Displays inputted JSON to the left, and info about the current variable being edited to the right */}
            {/* TextField for entering markdown, or markdown display if previewing */}
            {
                isPreviewOn ?
                    <Box sx={{
                        border: `1px solid ${palette.background.textPrimary}`,
                        color: isValueValid ? palette.background.textPrimary : palette.error.main,
                    }}>
                        {
                            previewMarkdown ?
                                <Markdown>{previewMarkdown}</Markdown> :
                                <p>{`Error: Invalid JSON - ${formik.values.format}`}</p>
                        }
                    </Box> :
                    <TextField
                        name="format"
                        disabled={!isEditing}
                        placeholder={"Enter JSON format. Click the '?' button for help."}
                        multiline
                        value={formik.values.format}
                        onChange={formik.handleChange}
                        style={{
                            minWidth: '-webkit-fill-available',
                            maxWidth: '-webkit-fill-available',
                            minHeight: '50px',
                            maxHeight: '800px',
                            background: 'transparent',
                            borderColor: formik.errors.format ? 'red' : 'unset',
                            borderRadius: '0 0 0.5rem 0.5rem',
                            borderTop: 'none',
                            fontFamily: 'inherit',
                            fontSize: 'inherit',
                        }}
                    />
            }
            {/* Bottom bar containing arrow buttons to switch to different incomplete/incorrect
             parts of the JSON, and an input for entering the currently-selected section of JSON */}
            {/* TODO */}
            {/* Helper text label */}
            {/* {
                helperText &&
                <Typography variant="body1" sx={{ color: 'red' }}>
                    {helperText}
                </Typography>
            } */}
        </Stack>
    );
}