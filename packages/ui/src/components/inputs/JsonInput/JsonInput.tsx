/**
 * Input for creating a JSON schema.
 */
import { useEffect, useMemo, useState } from 'react';
import { Box, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { JsonInputProps } from '../types';
import { HelpButton, StatusButton } from 'components/buttons';
import { isJson, jsonHelpText, jsonToMarkdown, Status, TERTIARY_COLOR } from 'utils';
import Markdown from 'markdown-to-jsx';
import {
    VisibilityOff as PreviewOffIcon,
    Visibility as PreviewOnIcon,
} from '@mui/icons-material';

export const JsonInput = ({
    id,
    disabled = false,
    error = false,
    format,
    helperText,
    minRows = 4,
    onChange,
    placeholder = '',
    value,
    variables,
}: JsonInputProps) => {
    const { palette } = useTheme();
    console.log('json input render', value);

    /**
     * If value not set, defaults to format with variables 
     * substituted for their known defaults.
     */
    useEffect(() => {
        if (disabled) return;
        if ((value ?? '').length > 0 || !format) return;
        // Initialize with stringified format
        let changedFormat: string = JSON.stringify(format);
        // If variables not set, return stringified format
        if (!variables) {
            onChange(changedFormat);
            return;
        }
        // Loop through variables and replace all instances of them in format
        // with their default values.
        for (const [variableName, variableData] of Object.entries(variables)) {
            // If variable has no default value, skip it
            if (variableData.defaultValue === undefined) continue;
            // Find locations of variables in format
            // TODO
        }
    }, [disabled, format, onChange, value, variables]);

    /**
     * Uses format, variables, and value
     */
    useEffect(() => {
        //TODO
    }, [format, variables]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const togglePreview = () => setIsPreviewOn(!isPreviewOn);
    const { previewMarkdown, isValueValid } = useMemo(() => ({
        previewMarkdown: jsonToMarkdown(value),
        isValueValid: isJson(value),
    }), [value]);

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
                    messages={isValueValid ? ['JSON is valid'] : ['JSON could not be parsed']}
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
                                <p>{`Error: Invalid JSON - ${value}`}</p>
                        }
                    </Box> :
                    <TextField
                        id={`markdown-input-${id}`}
                        disabled={disabled}
                        placeholder={placeholder}
                        rows={minRows}
                        value={value ?? ''}
                        onChange={(e) => { onChange(e.target.value) }}
                        style={{
                            minWidth: '-webkit-fill-available',
                            maxWidth: '-webkit-fill-available',
                            minHeight: '50px',
                            maxHeight: '800px',
                            background: 'transparent',
                            borderColor: error ? 'red' : 'unset',
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
            {
                helperText &&
                <Typography variant="body1" sx={{ color: 'red' }}>
                    {helperText}
                </Typography>
            }
        </Stack>
    );
}