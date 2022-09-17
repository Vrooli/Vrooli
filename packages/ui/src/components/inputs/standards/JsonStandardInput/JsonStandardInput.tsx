/**
 * Input for entering (and viewing format of) JSON data that 
 * must match a certain schema.
 */
import { JsonStandardInputProps } from '../types';
import { useState } from 'react';
import { Box, Stack, useTheme } from '@mui/material';
import { HelpButton } from 'components/buttons';
import { jsonHelpText } from 'utils';
import Editor from '@monaco-editor/react';

type JsonStandardInputState = {
    value: string,
    error?: {
        text: string,
        line: number,
    },
}

export const JsonStandardInput = ({
    defaultValue,
    isEditing,
}: JsonStandardInputProps) => {
    const { palette } = useTheme();

    const [state, setState] = useState<JsonStandardInputState>({
        value: '{\n\t"msg": "Type json here"\n}',
        error: undefined,
    });

    return (
        <Stack direction="column" spacing={0}>
            {/* Bar above TextField, for status and HelpButton */}
            <Box sx={{
                display: 'flex',
								justifyContent: 'space-between',
								alignItems: 'center',
                width: '100%',
                height: '56px',
                padding: '8px',
                background: palette.primary.light,
                color: palette.primary.contrastText,
                borderRadius: '8px 8px 0 0',
            }}>
							<p style={{ marginLeft: "8px" }}>{
								state.error ? `Error on line ${state.error.line}: {state.error.text}` : "No errors"
							}</p>
                <HelpButton
                    markdown={jsonHelpText}
                    sxRoot={{ marginRight: 1 }}
                />
            </Box>
            <Editor
                height="50vh"
                defaultLanguage="json"
                value={state.value}
                onChange={value => {
									console.log(value);
                	if (!value) {
										setState(prev => {
											return {
												value,
												error: { text: "JSON should not be empty", line: 1 },
											};
										});
										return;
									} else if (value.length >= 8192) {
										setState(prev => {
											return {
												value,
												error: { text: "JSON max length is 8192", line: 1 },
											};
										});
										return;
									}
									setState({
                  	value,
										error: undefined,
                  });
                }}
                onValidate={(markers) => {
									console.log(markers);
                    if (markers.length > 0) {
                        const marker = markers[0];
                        setState(prev => {
                            return { 
                                ...prev,
                                error: { text: marker.message, line: marker.startLineNumber },
                            };
                        });
                        return;
                    }
                }}
            />
        </Stack>
    );
}

