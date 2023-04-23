import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { SortIcon } from "@local/icons";
import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { labelledSortOptions } from "../../../utils/display/sorting";
import { SortMenu } from "../../lists/SortMenu/SortMenu";
import { searchButtonStyle } from "../styles";
export const SortButton = ({ options, setSortBy, sortBy, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [sortAnchorEl, setSortAnchorEl] = useState(null);
    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label, selected) => {
        setSortAnchorEl(null);
        if (selected)
            setSortBy(selected);
    };
    const sortOptionsLabelled = useMemo(() => labelledSortOptions(options), [options]);
    const sortByLabel = useMemo(() => t(sortBy, sortBy), [sortBy, t]);
    return (_jsxs(_Fragment, { children: [_jsx(SortMenu, { sortOptions: sortOptionsLabelled, anchorEl: sortAnchorEl, onClose: handleSortClose }), _jsx(Tooltip, { title: t("SortBy"), placement: "top", children: _jsxs(Box, { onClick: handleSortOpen, sx: searchButtonStyle(palette), children: [_jsx(SortIcon, { fill: palette.secondary.main }), _jsx(Typography, { variant: "body2", sx: { marginLeft: 0.5 }, children: sortByLabel })] }) })] }));
};
//# sourceMappingURL=SortButton.js.map