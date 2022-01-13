import { useCallback, useMemo } from 'react';
import { Box, Chip, FormControl, InputLabel, MenuItem, Select, Theme } from '@mui/material';
import isArray from 'lodash/isArray';
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
    className,
    style,
    ...props
}: SelectorProps) => {
    console.log('slelector', options, selected);
    const getOptionStyle = useCallback((theme: Theme, option: any) => {
        const label = getOptionLabel ? getOptionLabel(option) : option;
        return {
            fontWeight:
                options.find(o => (getOptionLabel ? getOptionLabel(o) : o) === label)
                    ? theme.typography.fontWeightRegular
                    : theme.typography.fontWeightMedium,
        };
    }, [options, getOptionLabel]);

    return (
        <FormControl
            variant="outlined"
            sx={{ width: fullWidth ? '-webkit-fill-available' : '' }}
        >
            <InputLabel
                id={inputAriaLabel}
                shrink={selected?.length > 0}
                sx={{ color: (t) => t.palette.background.textPrimary }}
            >
                {label}
            </InputLabel>
            <Select
                className={className}
                labelId={inputAriaLabel}
                value={selected}
                onChange={handleChange}
                label={label}
                required={required}
                disabled={disabled}
                multiple={multiple}
                renderValue={() => {
                    // If nothing is selected, don't display anything
                    if (!selected || (isArray(selected) && selected.length === 0)) {
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
                    ...style,
                    color: (t) => t.palette.background.textPrimary
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