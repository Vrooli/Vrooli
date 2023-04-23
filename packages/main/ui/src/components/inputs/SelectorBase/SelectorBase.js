import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon } from "@local/icons";
import { exists } from "@local/utils";
import { FormControl, FormHelperText, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
export function SelectorBase({ addOption, autoFocus = false, options, error, getOptionDescription, getOptionIcon, getOptionLabel, helperText, fullWidth = false, inputAriaLabel = "select-label", onBlur, onChange, name, noneOption = false, label = "Select", required = true, disabled = false, sx, tabIndex, value, }) {
    const { palette, typography } = useTheme();
    const { t } = useTranslation();
    const getOptionStyle = useCallback((label) => {
        const isSelected = exists(value) && getOptionLabel(value) === label;
        return {
            fontWeight: isSelected ? typography.fontWeightMedium : typography.fontWeightRegular,
        };
    }, [value, getOptionLabel, typography.fontWeightMedium, typography.fontWeightRegular]);
    const labels = useMemo(() => options.map((option) => {
        const labelText = getOptionLabel(option);
        const description = getOptionDescription ? getOptionDescription(option) : null;
        const Icon = getOptionIcon ? getOptionIcon(option) : null;
        return (_jsxs(MenuItem, { value: labelText, sx: { whiteSpace: "normal" }, children: [Icon && _jsx(ListItemIcon, { sx: {
                        minWidth: "32px",
                    }, children: _jsx(Icon, { fill: palette.background.textSecondary }) }), _jsxs(Stack, { direction: "column", children: [_jsx(ListItemText, { sx: {
                                ...getOptionStyle(labelText),
                                "& .MuiTypography-root": {
                                    fontWeight: description ? "bold" : typography.fontWeightRegular,
                                },
                            }, children: labelText }), description && _jsx(ListItemText, { sx: getOptionStyle(labelText), children: description })] })] }, labelText));
    }), [options, getOptionLabel, getOptionDescription, getOptionIcon, palette.background.textSecondary, getOptionStyle, typography.fontWeightRegular]);
    const findOption = useCallback((label) => options.find((option) => getOptionLabel(option) === label), [options, getOptionLabel]);
    const shrinkLabel = useMemo(() => {
        if (!exists(value))
            return false;
        return getOptionLabel(value)?.length > 0;
    }, [value, getOptionLabel]);
    return (_jsxs(FormControl, { variant: "outlined", error: error, sx: { width: fullWidth ? "-webkit-fill-available" : "" }, children: [_jsx(InputLabel, { id: inputAriaLabel, shrink: shrinkLabel, sx: { color: palette.background.textPrimary }, children: label }), _jsxs(Select, { "aria-describedby": `helper-text-${name}`, autoFocus: autoFocus, disabled: disabled, id: name, label: label, labelId: inputAriaLabel, name: name, onChange: (e) => { onChange(findOption(e.target.value)); }, onBlur: onBlur, required: required, value: exists(value) ? getOptionLabel(value) : "", variant: "outlined", sx: {
                    ...sx,
                    color: palette.background.textPrimary,
                    "& .MuiSelect-select": {
                        paddingTop: "12px",
                        paddingBottom: "12px",
                        display: "flex",
                        alignItems: "center",
                    },
                }, tabIndex: tabIndex, children: [noneOption ? (_jsx(MenuItem, { value: "", children: _jsx("em", { children: t("None") }) })) : null, labels, addOption ? (_jsxs(MenuItem, { value: "addOption", onClick: addOption.onSelect, children: [_jsx(AddIcon, { fill: palette.background.textPrimary, style: { marginRight: "8px" } }), _jsx("em", { children: addOption.label })] })) : null] }), helperText && _jsx(FormHelperText, { id: `helper-text-${name}`, children: helperText })] }));
}
//# sourceMappingURL=SelectorBase.js.map