import { useCallback, useMemo } from 'react';
import { FormControl, InputLabel, MenuItem, Select, useTheme } from '@mui/material';
import { SelectorProps } from '../types';

export function Selector<T extends string | number | { [x: string]: any }>({
    options,
    selected,
    getOptionLabel,
    handleChange,
    fullWidth = false,
    inputAriaLabel = 'select-label',
    noneOption = false,
    label = 'Select',
    required = true,
    disabled = false,
    color,
    sx,
    ...props
}: SelectorProps<T>) {
    const { palette, typography } = useTheme();

    // Render all labels
    const labels = useMemo(() => options.map((option) => getOptionLabel(option)), [options, getOptionLabel]);
    // Find option from label
    const findOption = useCallback((label: string) => options.find((option) => getOptionLabel(option) === label), [options, getOptionLabel]);

    const getOptionStyle = useCallback((label: string) => {
        const isSelected = selected && getOptionLabel(selected) === label;
        return {
            fontWeight: isSelected ? typography.fontWeightMedium : typography.fontWeightRegular,
        };
    }, [getOptionLabel, selected, typography.fontWeightMedium, typography.fontWeightRegular]);

    /**
     * Determines if label should be shrunk
     */
    const shrinkLabel = useMemo(() => {
        if (!selected) return false;
        return getOptionLabel(selected).length > 0;
    }, [selected, getOptionLabel]);

    return (
        <FormControl
            variant="outlined"
            sx={{ width: fullWidth ? '-webkit-fill-available' : '' }}
        >
            <InputLabel
                id={inputAriaLabel}
                shrink={shrinkLabel}
                sx={{ color: palette.background.textPrimary }}
            >
                {label}
            </InputLabel>
            <Select
                labelId={inputAriaLabel}
                value={selected ? getOptionLabel(selected) : ''}
                onChange={(e) => handleChange(findOption(e.target.value as string) as T, e)}
                label={label}
                required={required}
                disabled={disabled}
                variant="filled"
                {...props}
                sx={{
                    ...sx,
                    color: palette.background.textPrimary
                }}
            >
                {
                    noneOption ? (
                        <MenuItem value="">
                            <em>None</em>
                        </MenuItem>
                    ) : null
                }
                {
                    labels.map((label) => (
                        <MenuItem
                            key={label}
                            value={label}
                            style={getOptionStyle(label)}
                        >
                            {label}
                        </MenuItem>
                    ))
                }
            </Select>
        </FormControl>
    );
}