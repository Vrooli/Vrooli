import { useCallback, useMemo } from 'react';
import { Box, Chip, FormControl, InputLabel, MenuItem, Select, Theme, useTheme } from '@mui/material';
import { SelectorProps } from '../types';

export const Selector = ({
    options,
    getOptionLabel,
    selected,
    handleChange,
    fullWidth = false,
    multiple = false,
    inputAriaLabel = 'select-label',
    noneOption = false,
    label = 'Select',
    required = true,
    disabled = false,
    color,
    sx,
    ...props
}: SelectorProps) => {
    const { palette } = useTheme();

    const getOptionStyle = useCallback((theme: Theme, option: any) => {
        const label = getOptionLabel ? getOptionLabel(option) : option;
        return {
            fontWeight:
                options.find(o => (getOptionLabel ? getOptionLabel(o) : o) === label)
                    ? theme.typography.fontWeightRegular
                    : theme.typography.fontWeightMedium,
        };
    }, [options, getOptionLabel]);

    /**
     * Determines if label should be shrunk
     */
    const shrinkLabel = useMemo(() => {
        if (!selected) return false;
        if (typeof selected === 'string') return selected.length > 0;
        if (getOptionLabel && selected) return getOptionLabel(selected).length > 0;
        return false; 
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
                value={selected}
                onChange={handleChange}
                label={label}
                required={required}
                disabled={disabled}
                multiple={multiple}
                renderValue={() => {
                    // If nothing is selected, don't display anything
                    if (!selected || (Array.isArray(selected) && selected.length === 0)) {
                        return '';
                    }
                    // If not multiple, just display the selected option
                    if (!multiple) {
                        return getOptionLabel ? getOptionLabel(selected) : selected;
                    }
                    // If multiple, display all selected options as chips
                    return (
                        <Box
                            sx={{
                                display: 'flex',
                                flexWrap: 'wrap',
                            }}
                        >
                            {selected.map((o) => (
                                <Chip label={getOptionLabel ? getOptionLabel(o) : o} key={o} />
                            ))}
                        </Box>
                    )
                }}
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
                    options.map((o, index) => (
                        <MenuItem
                            key={`select-option-${index}`}
                            value={o}
                            sx={(t) => getOptionStyle(t, o)}
                        >
                            {getOptionLabel ? getOptionLabel(o) : o}
                        </MenuItem>
                    ))
                }
            </Select>
        </FormControl>
    );
}