import { FormControl, FormHelperText, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, useTheme } from '@mui/material';
import { useField } from 'formik';
import { useCallback, useMemo } from 'react';
import { SelectorProps } from '../types';

export function Selector<T extends string | number | { [x: string]: any }>({
    options,
    getOptionDescription,
    getOptionIcon,
    getOptionLabel,
    fullWidth = false,
    inputAriaLabel = 'select-label',
    onChange,
    name,
    noneOption = false,
    label = 'Select',
    required = true,
    disabled = false,
    sx,
}: SelectorProps<T>) {
    const { palette, typography } = useTheme();
    const [field, meta, helpers] = useField(name);

    const getOptionStyle = useCallback((label: string) => {
        const isSelected = field.value && getOptionLabel(field.value) === label;
        return {
            fontWeight: isSelected ? typography.fontWeightMedium : typography.fontWeightRegular,
        };
    }, [field.value, getOptionLabel, typography.fontWeightMedium, typography.fontWeightRegular]);

    // Render all labels
    const labels = useMemo(() => options.map((option) => {
        const labelText = getOptionLabel(option);
        const description = getOptionDescription ? getOptionDescription(option) : null;
        const Icon = getOptionIcon ? getOptionIcon(option) : null;
        return (
            <MenuItem key={labelText} value={labelText} sx={{ whiteSpace: 'normal' }}>
                {Icon && <ListItemIcon sx={{
                    minWidth: '32px',
                }}>
                    <Icon fill={palette.background.textSecondary} />
                </ListItemIcon>}
                <Stack direction="column">
                    <ListItemText sx={{
                        ...getOptionStyle(labelText),
                        // fontWeight: description ? 'bold!important' : typography.fontWeightRegular,
                        '& .MuiTypography-root': {
                            fontWeight: description ? 'bold' : typography.fontWeightRegular,
                        },
                    }}>{labelText}</ListItemText>
                    {description && <ListItemText sx={getOptionStyle(labelText)}>{description}</ListItemText>}
                </Stack>
            </MenuItem>
        );
    }), [options, getOptionLabel, getOptionDescription, getOptionIcon, palette.background.textSecondary, getOptionStyle, typography.fontWeightRegular]);

    // Find option from label
    const findOption = useCallback((label: string) => options.find((option) => getOptionLabel(option) === label), [options, getOptionLabel]);

    /**
     * Determines if label should be shrunk
     */
    const shrinkLabel = useMemo(() => {
        if (!field.value) return false;
        return getOptionLabel(field.value)?.length > 0;
    }, [field.value, getOptionLabel]);

    return (
        <FormControl
            variant="outlined"
            error={meta.touched && !!meta.error}
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
                aria-describedby={`helper-text-${name}`}
                disabled={disabled}
                id={name}
                label={label}
                labelId={inputAriaLabel}
                name={name}
                onChange={(e) => {
                    const valueAsT = findOption(e.target.value as string) as T;
                    field.onChange(valueAsT);
                    if (onChange) onChange(valueAsT);
                }}
                onBlur={field.onBlur}
                required={required}
                value={field.value ? getOptionLabel(field.value) : ''}
                variant="outlined"
                sx={{
                    ...sx,
                    color: palette.background.textPrimary,
                    '& .MuiSelect-select': {
                        paddingTop: '12px',
                        paddingBottom: '12px',
                        display: 'flex',
                        alignItems: 'center',
                    }
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
                    labels
                }
            </Select>
            {meta.touched && meta.error && <FormHelperText id={`helper-text-${name}`}>{meta.error}</FormHelperText>}
        </FormControl>
    );
}