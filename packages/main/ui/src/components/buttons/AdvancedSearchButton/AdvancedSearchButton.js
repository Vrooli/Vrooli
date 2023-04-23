import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BuildIcon } from "@local/icons";
import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, removeSearchParams, useLocation } from "../../../utils/route";
import { AdvancedSearchDialog } from "../../dialogs/AdvancedSearchDialog/AdvancedSearchDialog";
import { searchButtonStyle } from "../styles";
export const AdvancedSearchButton = ({ advancedSearchParams, advancedSearchSchema, searchType, setAdvancedSearchParams, zIndex, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (!advancedSearchSchema?.fields) {
            setAdvancedSearchParams(null);
            return;
        }
        if (typeof searchParams.advanced === "boolean")
            setAdvancedSearchDialogOpen(searchParams.advanced);
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        const allAdvancedSearchParams = advancedSearchSchema.fields.map(f => f.fieldName);
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema?.fields, setAdvancedSearchParams]);
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true); }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false);
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values) => {
        const valuesWithoutBlanks = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== 0));
        removeSearchParams(setLocation, advancedSearchSchema?.fields?.map(f => f.fieldName) ?? []);
        addSearchParams(setLocation, valuesWithoutBlanks);
        setAdvancedSearchParams(valuesWithoutBlanks);
    }, [advancedSearchSchema?.fields, setAdvancedSearchParams, setLocation]);
    useEffect(() => {
        addSearchParams(setLocation, { advanced: advancedSearchDialogOpen });
    }, [advancedSearchDialogOpen, setLocation]);
    return (_jsxs(_Fragment, { children: [_jsx(AdvancedSearchDialog, { handleClose: handleAdvancedSearchDialogClose, handleSearch: handleAdvancedSearchDialogSubmit, isOpen: advancedSearchDialogOpen, searchType: searchType, zIndex: zIndex + 1 }), advancedSearchParams && _jsx(Tooltip, { title: t("SeeAllSearchSettings"), placement: "top", children: _jsxs(Box, { onClick: handleAdvancedSearchDialogOpen, sx: searchButtonStyle(palette), children: [_jsx(BuildIcon, { fill: palette.secondary.main }), Object.keys(advancedSearchParams).length > 0 && _jsxs(Typography, { variant: "body2", sx: { marginLeft: 0.5 }, children: ["*", Object.keys(advancedSearchParams).length] })] }) })] }));
};
//# sourceMappingURL=AdvancedSearchButton.js.map