import { exists } from "@local/shared";
import { FormControl, FormHelperText, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, useTheme } from "@mui/material";
import { AddIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SelectorBaseProps } from "../types";

export function SelectorBase<T extends string | number | { [x: string]: any }>({
    addOption,
    autoFocus = false,
    options,
    error,
    getOptionDescription,
    getOptionIcon,
    getOptionLabel,
    helperText,
    fullWidth = false,
    inputAriaLabel = "select-label",
    onBlur,
    onChange,
    name,
    noneOption = false,
    label = "Select",
    required = true,
    disabled = false,
    sx,
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
        const labelText = getOptionLabel(option);
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
        return getOptionLabel(value)?.length > 0;
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
                sx={{ color: palette.background.textPrimary }}
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
                required={required}
                value={exists(value) ? getOptionLabel(value) : ""}
                variant="outlined"
                sx={{
                    ...sx,
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    "& .MuiSelect-select": {
                        paddingTop: "12px",
                        paddingBottom: "12px",
                        display: "flex",
                        alignItems: "center",
                    },
                }}
                tabIndex={tabIndex}
                // Don't show description
                renderValue={(value) => {
                    const option = findOption(value as string);
                    if (!exists(option)) return null;
                    const labelText = getOptionLabel(option);
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
                            <em>{t("None")}</em>
                        </MenuItem>
                    ) : null
                }
                {
                    labels
                }
                {
                    addOption ? (
                        <MenuItem value="addOption" onClick={addOption.onSelect}>
                            <AddIcon fill={palette.background.textPrimary} style={{ marginRight: "8px" }} />
                            <em>{addOption.label}</em>
                        </MenuItem>
                    ) : null
                }
            </Select>
            {helperText && <FormHelperText id={`helper-text-${name}`}>{helperText}</FormHelperText>}
        </FormControl>
    );
}
