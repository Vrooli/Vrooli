/**
 * Input for entering (and viewing format of) JSON data that 
 * must match a certain schema.
 */
import { JsonStandardInputProps } from '../types';
import { useState } from 'react';
import { Box, Stack, TextField, useTheme } from '@mui/material';
import { HelpButton, StatusButton } from 'components/buttons';
import { checkJsonErrors, jsonHelpText, Status } from 'utils';

interface JsonStandardInputState {
    errors: string[];
    isValid: boolean;
    value: string;
}

export const JsonStandardInput = ({
    defaultValue,
    isEditing,
}: JsonStandardInputProps) => {
    const { palette } = useTheme();

    const [state, setState] = useState<JsonStandardInputState>({
        errors: checkJsonErrors(defaultValue),
        isValid: checkJsonErrors(defaultValue).length === 0,
        value: defaultValue,
    });

    const onChange = event => {
        const errors = checkJsonErrors(event.target.value);
        setState({
            errors,
            isValid: errors.length === 0,
            value: event.target.value,
        });
    }

    return (
        <Stack direction="column" spacing={0}>
            {/* Bar above TextField, for status and HelpButton */}
            <Box sx={{
                display: 'flex',
                width: '100%',
                height: '56px',
                padding: '8px',
                background: palette.primary.light,
                color: palette.primary.contrastText,
                borderRadius: '8px 8px 0 0',
            }}>
                <StatusButton
                    status={state.isValid ? Status.Valid : Status.Invalid}
                    messages={state.errors.length === 0 ? ['JSON is valid'] : state.errors}
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
                sx={{
                    minWidth: '-webkit-fill-available',
                    maxWidth: '-webkit-fill-available',
                    minHeight: '50px',
                    maxHeight: '800px',
                    background: 'transparent',
                    fontFamily: 'inherit',
                    fontSize: 'inherit',
                    '& .MuiInputBase-root': {
                        borderColor: state.isValid ? 'unset' : 'red',
                        borderRadius: '0 0 8px 8px',
                    }
                }}
            />
        </Stack>
    );
}

