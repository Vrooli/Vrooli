/**
 * Input for entering (and viewing format of) JSON data that 
 * must match a certain schema.
 */
import { JsonStandardInputProps } from '../types';
import { useState } from 'react';
import { Box, Stack, TextField, useTheme } from '@mui/material';
import { HelpButton, StatusButton } from 'components/buttons';
import { isJson, jsonHelpText, jsonToString, Status } from 'utils';

export const JsonStandardInput = ({
    defaultValue,
    isEditing,
}: JsonStandardInputProps) => {
    const { palette } = useTheme();

    const [state, setState] = useState({
        value: defaultValue,
        valid: true,
    });

    const onChange = event => {
        if (isJson(event.target.value)) {
            setState({
                value: jsonToString(event.target.value) ?? '',
                valid: true,
            });
            return;
        }
        setState({
            value: event.target.value,
            valid: false,
        });
    }

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
                    status={state.valid ? Status.Valid : Status.Invalid}
                    messages={state.valid ? ['JSON is valid'] : ['JSON is empty or could not be parsed']}
                    sx={{
                        marginLeft: 1,
                        marginRight: 'auto',
                    }}
                />
                <HelpButton
                    markdown={jsonHelpText}
                    sxRoot={{ marginRight: 1 }}
                />
            </Box>
            {/* TextField for entering JSON */}
            <TextField
                name="format"
                disabled={!isEditing}
                placeholder={"Enter JSON format. Click the '?' button for help."}
                multiline
                value={state.value}
                onChange={onChange}
                style={{
                    minWidth: '-webkit-fill-available',
                    maxWidth: '-webkit-fill-available',
                    minHeight: '50px',
                    maxHeight: '800px',
                    background: 'transparent',
                    borderColor: state.valid ? 'unset' : 'red',
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
            {/* {
                helperText &&
                <Typography variant="body1" sx={{ color: 'red' }}>
                    {helperText}
                </Typography>
            } */}
        </Stack>
    );
}
