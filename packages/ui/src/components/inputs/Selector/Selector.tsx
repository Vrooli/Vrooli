import { exists } from "@local/shared";
import { FormControl, FormHelperText, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { type SelectorBaseProps, type SelectorProps } from "../types.js";

export function SelectorBase<T extends string | number | { [x: string]: any }>({
    addOption,
    autoFocus = false,
    options,
    error,
    getDisplayIcon,
    getOptionDescription,
    getOptionIcon,
    getOptionLabel,
    helperText,
    fullWidth = false,
    inputAriaLabel = "select-label",
    isRequired,
    onBlur,
    onChange,
    name,
    noneOption = false,
    noneText,
    label,
    disabled = false,
    sxs,
    tabIndex,
    value,
}: SelectorBaseProps<T>) {
    const { palette, typography, spacing } = useTheme();
    const { t } = useTranslation();

    const getOptionStyle = useCallback((label: string) => {
        const isSelected = exists(value) && getOptionLabel(value, t) === label;
        return {
            fontWeight: isSelected ? typography.fontWeightMedium : typography.fontWeightRegular,
        };
    }, [value, getOptionLabel, t, typography.fontWeightMedium, typography.fontWeightRegular]);

    const getLabelStyle = useCallback(function getLabelStyleCallback(label: string, description: string | null | undefined) {
        return {
            ...getOptionStyle(label),
            "& .MuiTypography-root": {
                fontWeight: description ? "bold" : typography.fontWeightRegular,
            },
        };
    }, [getOptionStyle, typography.fontWeightRegular]);

    // Render all labels
    const labels = useMemo(() => options.map((option) => {
        const labelText = getOptionLabel(option, t) ?? "";
        const description = getOptionDescription ? getOptionDescription(option, t) : null;
        const Icon = getOptionIcon ? getOptionIcon(option) : null;
        const labelStyle = getLabelStyle(labelText, description);

        return (
            <MenuItem key={labelText} value={labelText} sx={{ whiteSpace: "normal" }}>
                {
                    Icon ?
                        typeof Icon === "function" ?
                            <ListItemIcon sx={{ minWidth: "32px" }}>
                                <Icon fill={palette.background.textSecondary} />
                            </ListItemIcon> :
                            Icon :
                        null
                }
                <Stack direction="column">
                    <ListItemText sx={labelStyle}>{labelText}</ListItemText>
                    {description && <ListItemText sx={getOptionStyle(labelText)}>{description}</ListItemText>}
                </Stack>
            </MenuItem>
        );
    }), [options, getOptionLabel, t, getOptionDescription, getOptionIcon, getLabelStyle, palette.background.textSecondary, getOptionStyle]);

    // Find option from label
    const findOption = useCallback(function findOptionCallback(label: string) {
        return options.find((option) => getOptionLabel(option, t) === label);
    }, [options, getOptionLabel, t]);

    /**
     * Determines if label should be shrunk
     */
    const shrinkLabel = useMemo(() => {
        if (!exists(value)) return false;
        const labelText = getOptionLabel(value, t);
        return typeof labelText === "string" && labelText.length > 0;
    }, [value, getOptionLabel, t]);

    const renderValue = useCallback(function renderValueCallback(value: string) {
        const option = findOption(value as string);
        if (!exists(option)) return null;
        const labelText = getOptionLabel(option, t) ?? "";
        // Icon function can be overridden by getDisplayIcon
        const Icon = typeof getDisplayIcon === "function"
            ? getDisplayIcon(option) :
            typeof getOptionIcon === "function"
                ? getOptionIcon(option) :
                null;
        return (
            <Stack direction="row" alignItems="center">
                {
                    Icon ?
                        typeof Icon === "function" ?
                            <ListItemIcon sx={{ minWidth: "32px" }}>
                                <Icon fill={palette.background.textSecondary} />
                            </ListItemIcon> :
                            Icon :
                        null
                }
                <ListItemText sx={getOptionStyle(labelText)}>{labelText}</ListItemText>
                {/* Note that we omit the description */}
            </Stack>
        );
    }, [findOption, getDisplayIcon, getOptionIcon, getOptionLabel, getOptionStyle, palette.background.textSecondary, t]);

    const selectStyle = useMemo(function selectStyle() {
        return {
            ...sxs?.root,
            borderRadius: spacing(3),
            color: palette.background.textPrimary,
            backgroundColor: palette.background.paper,
            "& .MuiSelect-select": {
                padding: "8px",
                display: "flex",
                alignItems: "center",
            },
            "& .MuiSelect-icon": {
                color: "inherit",
            },
            "& .MuiOutlinedInput-notchedOutline": {
                border: "none",
            },
            "& .MuiPaper-root": {
                zIndex: 9999,
            },
            "& fieldset": {
                ...sxs?.fieldset,
                border: "none",
            },
            "& svg": {
                display: "flex",
                alignItems: "center",
            },
        };
    }, [sxs, spacing, palette.background.paper, palette.background.textPrimary]);

    return (
        <FormControl
            variant="outlined"
            error={error}
            sx={{
                width: fullWidth ? "-webkit-fill-available" : "",
                // Adjust label position when not shrunk to align with internal padding
                "& .MuiInputLabel-root:not(.MuiInputLabel-shrink)": {
                    transform: "translate(14px, 8px) scale(1)",
                },
            }}
        >
            <InputLabel
                id={inputAriaLabel}
                shrink={shrinkLabel}
            >
                {label}
            </InputLabel>
            <Select
                aria-describedby={`helper-text-${name}`}
                autoFocus={autoFocus}
                disabled={disabled}
                id={name}
                label={label}
                labelId={inputAriaLabel}
                name={name}
                onChange={(e) => { onChange(findOption(e.target.value as string) as T); }}
                onBlur={onBlur}
                required={isRequired}
                value={exists(value) ? (getOptionLabel(value, t) ?? undefined) : ""}
                variant="outlined"
                sx={selectStyle}
                tabIndex={tabIndex}
                renderValue={renderValue}
            >
                {
                    noneOption ? (
                        <MenuItem value="">
                            <em>{noneText || t("None")}</em>
                        </MenuItem>
                    ) : null
                }
                {
                    labels
                }
                {
                    addOption ? (
                        <MenuItem
                            aria-label={addOption.label}
                            value="addOption"
                            onClick={addOption.onSelect}
                        >
                            <IconCommon
                                decorative
                                fill="inherit"
                                name="Add"
                                style={{ marginRight: "8px" }}
                            />
                            <em>{addOption.label}</em>
                        </MenuItem>
                    ) : null
                }
            </Select>
            {helperText && <FormHelperText id={`helper-text-${name}`}>{typeof helperText === "string" ? helperText : JSON.stringify(helperText)}</FormHelperText>}
        </FormControl>
    );
}

export function Selector<T extends string | number | { [x: string]: any }>({
    name,
    onChange,
    ...props
}: SelectorProps<T>) {
    const [field, meta, helpers] = useField(name);

    const handleChange = useCallback(function handleChangeCallback(newValue: T) {
        if (onChange) onChange(newValue);
        helpers.setValue(newValue);
    }, [onChange, helpers]);

    return (
        <SelectorBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
}
