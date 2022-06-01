/**
 * Input for creating a JSON schema.
 */
import { useEffect, useMemo, useState } from 'react';
import { Box, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { JsonInputProps } from '../types';
import { HelpButton, StatusButton } from 'components/buttons';
import { Status } from 'utils';
import Markdown from 'markdown-to-jsx';
import {
    VisibilityOff as PreviewOffIcon,
    Visibility as PreviewOnIcon,
} from '@mui/icons-material';

const helpText =
    `JSON is a widely used format for storing data objects.  

If you are unfamiliar with JSON, you may [read this guide](https://www.tutorialspoint.com/json/json_quick_guide.htm) to learn how to use it.  

On top of the JSON format, we also support the following:  
- **Variables**: Any key or value can be substituted with a variable. These are used to make it easy for users to fill in their own data, as well as 
provide details about those parts of the JSON. Variables follow the format of &lt;variable_name&gt;.  
- **Optional fields**: Fields that are not required can be marked as optional. Optional fields must start with a question mark (e.g. ?field_name).  
- **Additional fields**: If an object contains arbitrary fields, add a field with brackets (e.g. [variable]).  
- **Data types**: Data types are specified by reserved words (e.g. &lt;string&gt;, &lt;number | boolean&gt;, &lt;any&gt;, etc.), variable names (e.g. &lt;variable_name&gt;), standard IDs (e.g. &lt;decdf6c8-4230-4777-b8e3-799df2503c42&gt;), or simply entered as normal JSON.
`

const TERTIARY_COLOR = '#95f3cd';

type Position = { start: number; end: number };
type VariablePosition = { field: Position; value: Position };

/**
 * Checks if a string can be parsed as JSON.
 * @param value The string to check.
 * @returns True if the string can be parsed as JSON, false otherwise.
 */
const isJson = (value: string | null): boolean => {
    try {
        value && JSON.parse(value);
        return true;
    } catch (e) {
        return false;
    }
};

/**
 * Converts a JSON string to a pretty-printed JSON markdown string.
 * @param value The JSON string to convert.
 * @returns The pretty-printed JSON string, to be rendered in <Markdown />.
 */
const jsonToMarkdown = (value: string | null): string | null => {
    try {
        return value ? `\`\`\`json\n${JSON.stringify(JSON.parse(value), null, 2)}\n\`\`\`` : null;
    } catch (e) {
        return null;
    }
}

/**
 * Finds the start and end positions of every matching variable/value pair in a JSON format
 * @param variable Name of variable
 * @param format format to search
 * @returns Array of start and end positions of every matching variable/value pair
 */
const findVariablePositions = (variable: string, format: { [x: string]: any | undefined }): VariablePosition[] => {
    // Initialize array of variable positions
    const variablePositions: VariablePosition[] = [];
    // Check for valid format
    if (!format) return variablePositions;
    const formatString = JSON.stringify(format);
    // Create substring to search for
    const keySubstring = `"<${variable}>":`;
    // Create fields to store data about the current position in the format string
    let openBracketCounter = 0; // Net number of open brackets since last variable
    let inQuotes = false; // If in quotes, ignore brackets
    let index = 1; // Current index in format string. Ignore first character, since we check the previous character in the loop (and the first character will always be a bracket)
    let keyStartIndex = -1; // Start index of variable's key
    let valueStartIndex = -1; // Start index of variable's value
    // Loop through format string, until end is reached. 
    while (index < formatString.length) {
        const currChar = formatString[index];
        const lastChar = formatString[index - 1];
        // If current and last char are "\", then the next character is escaped and should be ignored
        if (currChar !== '\\' && lastChar !== '\\') {
            // If variableStart has not been found, then check if we are at the start of the next variable instance
            if (keyStartIndex === -1) {
                console.log('KEY A', currChar, lastChar, keyStartIndex, valueStartIndex)
                // If the rest of the formatString is long enough to contain the variable substring, 
                // and the substring is found, then we are at the start of a new variable instance
                if (formatString.length - index >= keySubstring.length && formatString.substring(index, index + keySubstring.length) === keySubstring) {
                    keyStartIndex = index;
                }
            }
            // If variableStart has been found, check for the start of the value
            else if (index === keyStartIndex + keySubstring.length) {
                console.log('KEY B', currChar, lastChar, keyStartIndex, valueStartIndex)
                // If the current character is not a quote or an open bracket, then the value is invalid
                if (currChar !== '"' && currChar !== '{') {
                    keyStartIndex = -1;
                    valueStartIndex = -1;
                }
                // Otherwise, set up the startIndex, bracket counter and inQuotes flag
                valueStartIndex = index;
                openBracketCounter = Number(currChar === '{');
                inQuotes = currChar === '"';
            }
            // If variableStart has been found, check if we are at the end
            else if (index > keyStartIndex + keySubstring.length) {
                openBracketCounter += Number(currChar === '{');
                openBracketCounter -= Number(currChar === '}');
                if (currChar === '"') inQuotes = !inQuotes;
                // If the bracket counter is 0, and we are not in quotes, then we are at the end of the variable
                if (openBracketCounter === 0 && !inQuotes) {
                    // Add the variable to the array
                    variablePositions.push({
                        field: {
                            start: keyStartIndex,
                            end: keyStartIndex + keySubstring.length,
                        },
                        value: {
                            start: valueStartIndex,
                            end: index,
                        },
                    });
                }
            }
        } else index++;
        index++;
    }
    return variablePositions;
}

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
    console.log("JSON INPUT", value)

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
                    markdown={helpText}
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
                        color: isValueValid ? palette.primary.contrastText : palette.error.main,
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