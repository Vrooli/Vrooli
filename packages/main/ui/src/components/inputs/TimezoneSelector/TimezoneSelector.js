import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ArrowDropDownIcon, ArrowDropUpIcon } from "@local/icons";
import { IconButton, ListItem, Popover, Stack, TextField, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { FixedSizeList } from "react-window";
import { MenuTitle } from "../../dialogs/MenuTitle/MenuTitle";
const formatOffset = (offset) => {
    const sign = offset > 0 ? "-" : "+";
    const hours = Math.abs(Math.floor(offset / 60));
    const minutes = Math.abs(offset % 60);
    return `${sign}${hours.toString().padStart(2, "0")}:${minutes.toString().padStart(2, "0")}`;
};
const getTimezoneOffset = (timezone) => {
    const now = new Date();
    const localTime = now.getTime();
    const localOffset = now.getTimezoneOffset() * 60 * 1000;
    const utcTime = localTime + localOffset;
    const targetTime = new Date(now.toLocaleString("en-US", { timeZone: timezone }));
    const targetOffset = targetTime.getTime() - utcTime;
    const roundedOffset = Math.round((targetOffset / (60 * 1000)) * 10) / 10;
    return -roundedOffset;
};
export const TimezoneSelector = ({ onChange, ...props }) => {
    const { palette } = useTheme();
    const [field, , helpers] = useField(props.name);
    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((event) => {
        setSearchString(event.target.value);
    }, []);
    const timezones = useMemo(() => {
        const allTimezones = Intl.supportedValuesOf("timeZone");
        if (searchString.length > 0) {
            return allTimezones.filter(tz => tz.toLowerCase().includes(searchString.toLowerCase()));
        }
        return allTimezones;
    }, [searchString]);
    const timezoneData = useMemo(() => {
        return timezones.map((timezone) => {
            const timezoneOffset = getTimezoneOffset(timezone);
            const formattedOffset = formatOffset(timezoneOffset);
            return { timezone, timezoneOffset, formattedOffset };
        });
    }, [timezones]);
    const [anchorEl, setAnchorEl] = useState(null);
    const open = Boolean(anchorEl);
    const onOpen = useCallback((event) => {
        setAnchorEl(event.currentTarget);
    }, []);
    const onClose = useCallback(() => {
        setSearchString("");
        setAnchorEl(null);
    }, []);
    return (_jsxs(_Fragment, { children: [_jsxs(Popover, { open: open, anchorEl: anchorEl, onClose: onClose, anchorOrigin: {
                    vertical: "bottom",
                    horizontal: "center",
                }, transformOrigin: {
                    vertical: "top",
                    horizontal: "center",
                }, sx: {
                    "& .MuiPopover-paper": {
                        width: "100%",
                        maxWidth: 500,
                    },
                }, children: [_jsx(MenuTitle, { ariaLabel: "", title: "Select Timezone", onClose: onClose }), _jsxs(Stack, { direction: "column", spacing: 2, p: 2, children: [_jsx(TextField, { placeholder: "Enter timezone...", autoFocus: true, value: searchString, onChange: updateSearchString }), _jsx(FixedSizeList, { height: 600, width: "100%", itemSize: 46, itemCount: timezones.length, overscanCount: 5, children: ({ index, style }) => {
                                    const { timezone, formattedOffset } = timezoneData[index];
                                    return (_jsx(ListItem, { button: true, style: style, onClick: () => {
                                            helpers.setValue(timezone);
                                            onChange?.(timezone);
                                            setSearchString("");
                                        }, children: _jsxs(Stack, { direction: "row", justifyContent: "space-between", alignItems: "center", width: "100%", children: [_jsx(Typography, { children: timezone }), _jsx(Typography, { children: ` (UTC${formattedOffset})` })] }) }, index));
                                } })] })] }), _jsx(TextField, { ...props, value: field.value, variant: "outlined", onClick: onOpen, InputProps: {
                    endAdornment: (_jsx(IconButton, { size: "small", "aria-label": "timezone-select", children: open ?
                            _jsx(ArrowDropUpIcon, { fill: palette.background.textPrimary }) :
                            _jsx(ArrowDropDownIcon, { fill: palette.background.textPrimary }) })),
                } })] }));
};
//# sourceMappingURL=TimezoneSelector.js.map