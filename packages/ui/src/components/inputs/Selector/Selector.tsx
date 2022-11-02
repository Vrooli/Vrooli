import { useCallback, useMemo } from 'react';
import { FormControl, InputLabel, MenuItem, Select, useTheme } from '@mui/material';
import { SelectorProps } from '../types';

export function Selector<T extends string | number | { [x: string]: any }>({
    options,
    selected,
    getOptionLabel,
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
}: SelectorProps<T>) {
    const { palette, typography } = useTheme();

    const getOptionStyle = useCallback((option: T) => {
        const isSelected = selected && getOptionLabel(selected) === getOptionLabel(option);
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
                value={selected}
                onChange={handleChange as any}
                label={label}
                required={required}
                disabled={disabled}
                multiple={multiple}
                variant="filled"
                renderValue={() => (<span>{selected ? getOptionLabel(selected) : ''}</span>)}
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
                            value={getOptionLabel(o)}
                            sx={getOptionStyle(o)}
                        >
                            {getOptionLabel(o)}
                        </MenuItem>
                    ))
                }
            </Select>
        </FormControl>
    );
}