import { createElement as _createElement } from "react";
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { SearchIcon } from "@local/icons";
import { Autocomplete, IconButton, Input, ListItemText, MenuItem, Paper, Popper, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { findSearchResults, shapeSearchText } from "../../../../utils/search/siteToSearch";
const FullWidthPopper = function (props) {
    return _jsx(Popper, { ...props, style: {
            width: "fit-content",
            maxWidth: "100%",
        }, placement: "bottom-start" });
};
export const SettingsSearchBar = ({ debounce = 200, id = "search-bar", options = [], onChange, onInputChange, placeholder = "Search...", value, ...props }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [internalValue, setInternalValue] = useState(value);
    const [highlightedOption, setHighlightedOption] = useState(null);
    const handleChange = useCallback((event) => {
        const { value } = event.target;
        setInternalValue(value);
        setHighlightedOption(null);
        onChange(value);
    }, [onChange]);
    const onHighlightChange = useCallback((event, option, reason) => {
        if (option && option.label && reason === "keyboard") {
            setHighlightedOption(option);
        }
    }, []);
    const handleSelect = useCallback((option) => {
        onInputChange(option);
    }, [onInputChange]);
    const onSubmit = useCallback((event, value, reason, details) => {
        if (highlightedOption) {
            handleSelect(highlightedOption);
        }
    }, [highlightedOption, handleSelect]);
    const onKeyDown = useCallback((event) => {
        if (event.key === "ArrowRight" && highlightedOption) {
            setInternalValue(highlightedOption.label);
            onChange("");
        }
    }, [highlightedOption, onChange]);
    return (_jsx(Autocomplete, { disablePortal: true, id: id, filterOptions: findSearchResults, options: options, onHighlightChange: onHighlightChange, onKeyDown: onKeyDown, inputValue: internalValue, getOptionLabel: (option) => option.label ?? "", onSubmit: (event) => {
            event.preventDefault();
            event.stopPropagation();
        }, onChange: onSubmit, PopperComponent: FullWidthPopper, renderOption: (props, option) => {
            const shapedSearchString = shapeSearchText(internalValue);
            let displayedKeyword = "";
            if (option.keywords && shapedSearchString.length > 0) {
                for (let i = 0; i < option.keywords?.length; i++) {
                    const keyword = option.keywords[i];
                    const unshapedKeyword = option.unshapedKeywords[i];
                    if (unshapedKeyword === option.label || keyword.includes(shapedSearchString))
                        continue;
                    if (keyword.includes(shapedSearchString)) {
                        displayedKeyword = unshapedKeyword;
                        break;
                    }
                }
            }
            return (_createElement(MenuItem, { ...props, key: option.value, onClick: () => {
                    const label = option.label ?? "";
                    setInternalValue(label);
                    onChange(label);
                    handleSelect(option);
                } },
                _jsx(ListItemText, { sx: {
                        "& .MuiTypography-root": {
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                            overflow: "hidden",
                        },
                    }, children: option.label }),
                displayedKeyword && (_jsx(ListItemText, { sx: {
                        "& .MuiTypography-root": {
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                            overflow: "hidden",
                        },
                    }, children: displayedKeyword }))));
        }, renderInput: (params) => (_jsxs(Paper, { component: "form", sx: {
                p: "2px 4px",
                display: "flex",
                alignItems: "center",
                borderRadius: "10px",
            }, children: [_jsx(Input, { id: params.id, disabled: params.disabled, disableUnderline: true, fullWidth: params.fullWidth, value: internalValue, onChange: handleChange, placeholder: placeholder, autoFocus: props.autoFocus ?? false, inputProps: params.inputProps, ref: params.InputProps.ref, size: params.size, sx: {
                        ml: 1,
                        flex: 1,
                        "& .MuiAutocomplete-endAdornment": {
                            width: "48px",
                            height: "48px",
                            top: "0",
                            position: "relative",
                            "& .MuiButtonBase-root": {
                                width: "48px",
                                height: "48px",
                            },
                        },
                    } }), _jsx(IconButton, { sx: {
                        width: "48px",
                        height: "48px",
                    }, "aria-label": "main-search-icon", children: _jsx(SearchIcon, { fill: palette.background.textSecondary }) })] })), noOptionsText: t("NoResults", { ns: "error" }), sx: {
            "& .MuiAutocomplete-inputRoot": {
                paddingRight: "0 !important",
            },
        } }));
};
//# sourceMappingURL=SettingsSearchBar.js.map