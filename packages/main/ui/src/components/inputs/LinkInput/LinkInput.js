import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { SearchIcon } from "@local/icons";
import { Box, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "../../../utils/display/listTools";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog";
export const LinkInput = ({ label, name = "link", zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const textFieldRef = useRef(null);
    const [field, , helpers] = useField(name);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedUrl) => {
        setSearchOpen(false);
        if (selectedUrl) {
            helpers.setValue(selectedUrl);
        }
    }, [helpers]);
    const { title, subtitle } = useMemo(() => {
        if (!field.value)
            return {};
        if (field.value.startsWith(window.location.origin)) {
            const displayDataJson = localStorage.getItem(`objectFromUrl:${field.value}`);
            let displayData;
            try {
                displayData = displayDataJson ? JSON.parse(displayDataJson) : null;
            }
            catch (e) { }
            if (displayData) {
                let { title, subtitle } = getDisplay(displayData);
                if (subtitle && subtitle.length > 100) {
                    subtitle = subtitle.substring(0, 100) + "...";
                }
                return { title, subtitle };
            }
        }
        return {};
    }, [field.value]);
    return (_jsxs(_Fragment, { children: [_jsx(FindObjectDialog, { find: "Url", isOpen: searchOpen, handleCancel: closeSearch, handleComplete: closeSearch, zIndex: zIndex + 1 }), _jsxs(Box, { children: [_jsxs(Stack, { direction: "row", spacing: 0, children: [_jsx(Field, { fullWidth: true, name: name, label: label ?? t("Link"), as: TextField, ref: textFieldRef, sx: {
                                    "& .MuiInputBase-root": {
                                        borderRadius: "5px 0 0 5px",
                                    },
                                } }), _jsx(ColorIconButton, { "aria-label": 'find URL', onClick: openSearch, background: palette.secondary.main, sx: {
                                    borderRadius: "0 5px 5px 0",
                                    height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                                }, children: _jsx(SearchIcon, {}) })] }), title && (_jsx(Tooltip, { title: subtitle, children: _jsxs(Typography, { variant: 'body2', ml: 1, children: [title, subtitle ? " - " + subtitle : ""] }) }))] })] }));
};
//# sourceMappingURL=LinkInput.js.map