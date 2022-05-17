/**
 * TextField for entering (and viewing format of) JSON.
 */
import { useEffect } from 'react';
import { Box, Stack, Typography, useTheme } from '@mui/material';
import { JSONInputProps } from '../types';
import { Pubs } from 'utils';
import { HelpButton } from 'components/buttons';

const TERTIARY_COLOR = '#95f3cd';

export const JSONInput = ({
    id,
    description,
    disabled = false,
    error = false,
    format,
    helperText,
    minRows = 4,
    onChange,
    placeholder = '',
    title = 'JSON Input',
    value,
    variables,
}: JSONInputProps) => {
    const { palette } = useTheme();

    /**
     * If value not set, defauls to format with variables 
     * substituted for their known defaults.
     */
    useEffect(() => {
        if (value.length > 0 || !format) return;
        // First make sure format is valid JSON
        let formatJSON;
        try {
            formatJSON = JSON.parse(format);
        }
        catch (error) {
            PubSub.publish(Pubs.Snack, { message: 'Format supplied to JSON input is invalid JSON.', severity: 'error' });
            console.error(error);
            return;
        }
        let changedValue = formatJSON;
        // If variables not set, return format as is
        if (!variables) {
            onChange(formatJSON);
            return;
        }
        // Loop through variables and replace all instances of them in format
        // with their default values.
        for (const [variableName, variableData] of Object.entries(variables)) {
            // If variable has no default value, skip it
            if (variableData.defaultValue === undefined) continue;
            // 
        }
    }, [format, onChange, variables]);

    /**
     * Uses format, variables, and value
     */
    useEffect(() => {
        //TODO
    }, [format, variables]);

    return (
        <Stack direction="column" spacing={0}>
            {/* Bar above TextField, for title and HelpButton */}
            <Box sx={{
                display: 'flex',
                width: '100%',
                padding: '0.5rem',
                borderBottom: '1px solid #e0e0e0',
                background: palette.primary.light,
                color: palette.primary.contrastText,
                borderRadius: '0.5rem 0.5rem 0 0',
            }}>
                <Typography variant="h6">{title}</Typography>
                {description && <HelpButton
                    markdown={description}
                    sxRoot={{ margin: "auto", marginRight: 1, marginLeft: 'auto' }}
                    sx={{ color: TERTIARY_COLOR }}
                />}
            </Box>
            {/* Displays inputted JSON to the left, and info about the current variable being edited to the right */}
            {/* TextField for entering markdown, or markdown display if previewing */}
            <textarea
                id={`markdown-input-${id}`}
                disabled={disabled}
                placeholder={placeholder}
                rows={minRows}
                value={value}
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