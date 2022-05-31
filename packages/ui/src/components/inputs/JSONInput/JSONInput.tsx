/**
 * Input for creating a JSON schema.
 */
import { useEffect } from 'react';
import { Box, Stack, Typography, useTheme } from '@mui/material';
import { JsonInputProps } from '../types';
import { HelpButton } from 'components/buttons';

const TERTIARY_COLOR = '#95f3cd';

type Position = { start: number; end: number };
type VariablePosition = { field: Position; value: Position };
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
}: JsonInputProps) => {
    const { palette } = useTheme();

    /**
     * If value not set, defaults to format with variables 
     * substituted for their known defaults.
     */
    useEffect(() => {
        if (value.length > 0 || !format) return;
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
    }, [format, onChange, value.length, variables]);

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