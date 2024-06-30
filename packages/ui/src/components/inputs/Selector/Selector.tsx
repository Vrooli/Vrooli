import { exists } from "@local/shared";
import { FormControl, FormHelperText, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, useTheme } from "@mui/material";
import { useField } from "formik";
import { AddIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SelectorBaseProps, SelectorProps } from "../types";

export function SelectorBase<T extends string | number | { [x: string]: any }>({
    addOption,
    autoFocus = false,
    color,
    options,
    error,
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
    label = "Select",
    disabled = false,
    sxs,
    tabIndex,
    value,
}: SelectorBaseProps<T>) {
    const { palette, typography } = useTheme();
    const { t } = useTranslation();

    const getOptionStyle = useCallback((label: string) => {
        const isSelected = exists(value) && getOptionLabel(value) === label;
        return {
            fontWeight: isSelected ? typography.fontWeightMedium : typography.fontWeightRegular,
        };
    }, [value, getOptionLabel, typography.fontWeightMedium, typography.fontWeightRegular]);

    // Render all labels
    const labels = useMemo(() => options.map((option) => {
        const labelText = getOptionLabel(option) ?? "";
        const description = getOptionDescription ? getOptionDescription(option) : null;
        const Icon = getOptionIcon ? getOptionIcon(option) : null;
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
                    <ListItemText sx={{
                        ...getOptionStyle(labelText),
                        // fontWeight: description ? 'bold!important' : typography.fontWeightRegular,
                        "& .MuiTypography-root": {
                            fontWeight: description ? "bold" : typography.fontWeightRegular,
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
        if (!exists(value)) return false;
        const labelText = getOptionLabel(value);
        return typeof labelText === "string" && labelText.length > 0;
    }, [value, getOptionLabel]);

    return (
        <FormControl
            variant="outlined"
            error={error}
            sx={{ width: fullWidth ? "-webkit-fill-available" : "" }}
        >
            <InputLabel
                id={inputAriaLabel}
                shrink={shrinkLabel}
                sx={{ color: color ?? palette.background.textPrimary }}
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
                value={exists(value) ? getOptionLabel(value) : ""}
                variant="outlined"
                sx={{
                    ...sxs?.root,
                    color: color ?? palette.background.textPrimary,
                    "& .MuiSelect-select": {
                        paddingTop: "12px",
                        paddingBottom: "12px",
                        display: "flex",
                        alignItems: "center",
                    },
                    "& .MuiSelect-icon": {
                        color: color ?? palette.background.textPrimary,
                    },
                    "& .MuiOutlinedInput-notchedOutline": {
                        borderColor: color ?? palette.background.textPrimary,
                    },
                    "& .MuiPaper-root": {
                        zIndex: 9999,
                    },
                    "& fieldset": {
                        ...sxs?.fieldset,
                    },
                }}
                tabIndex={tabIndex}
                // Don't show description
                renderValue={(value) => {
                    const option = findOption(value as string);
                    if (!exists(option)) return null;
                    const labelText = getOptionLabel(option) ?? "";
                    const Icon = getOptionIcon ? getOptionIcon(option) : null;
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
                        </Stack>
                    );
                }}
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
                        <MenuItem value="addOption" onClick={addOption.onSelect}>
                            <AddIcon fill={color ?? palette.background.textPrimary} style={{ marginRight: "8px" }} />
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

    return (
        <SelectorBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={(newValue) => {
                if (onChange) onChange(newValue);
                helpers.setValue(newValue);
            }}
        />
    );
}
