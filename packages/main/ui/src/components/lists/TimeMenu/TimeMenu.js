import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Menu, MenuItem } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DateRangeMenu } from "../DateRangeMenu/DateRangeMenu";
const timeOptions = {
    "All Time": undefined,
    "Past Year": 31536000000,
    "Past Month": 2592000000,
    "Past Week": 604800000,
    "Past 24 Hours": 86400000,
    "Past Hour": 3600000,
};
export function TimeMenu({ anchorEl, onClose, }) {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);
    const [customRangeAnchorEl, setCustomRangeAnchorEl] = useState(null);
    const handleTimeOpen = (event) => setCustomRangeAnchorEl(event.currentTarget);
    const handleTimeClose = () => {
        setCustomRangeAnchorEl(null);
    };
    const menuItems = useMemo(() => Object.keys(timeOptions).map((label) => (_jsx(MenuItem, { value: timeOptions[label], onClick: () => {
            if (!timeOptions[label])
                onClose(label);
            else
                onClose(label.replace("Past ", ""), { after: new Date(Date.now() - timeOptions[label]) });
        }, children: label }, label))), [onClose]);
    return (_jsxs(Menu, { id: "results-time-menu", anchorEl: anchorEl, disableScrollLock: true, open: open, onClose: () => onClose(), MenuListProps: { "aria-labelledby": "results-time-menu-list" }, children: [menuItems, _jsx(MenuItem, { id: 'custom-range-menu-item', value: 'custom', onClick: handleTimeOpen, children: t("CustomRange") }), _jsx(DateRangeMenu, { anchorEl: customRangeAnchorEl, onClose: handleTimeClose, onSubmit: (after, before) => onClose("Custom", { after, before }) })] }));
}
//# sourceMappingURL=TimeMenu.js.map