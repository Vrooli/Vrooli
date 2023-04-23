import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Button, Grid, Popover, Stack, TextField } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { fromDatetimeLocal, toDatetimeLocal } from "../../../utils/shape/general";
import { TopBar } from "../../navigation/TopBar/TopBar";
export const DateRangeMenu = ({ anchorEl, minDate, maxDate, onClose, onSubmit, range, strictIntervalRange, }) => {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);
    const [after, setAfter] = useState(range?.after ?? minDate);
    const [before, setBefore] = useState(range?.before ?? maxDate);
    const handleAfterChange = useCallback((date) => { setAfter(date ?? minDate); }, [minDate]);
    const handleBeforeChange = useCallback((date) => { setBefore(date ?? maxDate); }, [maxDate]);
    useEffect(() => {
        setAfter(range?.after ?? minDate);
        setBefore(range?.before ?? maxDate);
    }, [range, minDate, maxDate]);
    useEffect(() => {
        let changedAfter = after ?? minDate ?? new Date();
        if (changedAfter < (minDate ?? 0)) {
            changedAfter = minDate ?? new Date(0);
        }
        const latestBefore = new Date(Date.now() - (strictIntervalRange ?? 0));
        if (changedAfter > latestBefore) {
            changedAfter = latestBefore;
        }
        if (changedAfter.getTime() - (after?.getTime() ?? 0) > 1000) {
            setAfter(changedAfter);
            if (strictIntervalRange && before !== new Date(changedAfter.getTime() + strictIntervalRange)) {
                setBefore(new Date(changedAfter.getTime() + strictIntervalRange));
            }
        }
    }, [after, before, minDate, strictIntervalRange]);
    useEffect(() => {
        let changedBefore = before ?? maxDate ?? new Date();
        if (after && changedBefore < after) {
            changedBefore = after;
        }
        if (changedBefore > new Date()) {
            changedBefore = new Date();
        }
        if (changedBefore.getTime() !== (before?.getTime() ?? 0)) {
            setBefore(changedBefore);
        }
    }, [after, before, maxDate]);
    return (_jsxs(Popover, { id: 'date-range-popover', anchorEl: anchorEl, open: open, onClose: onClose, disableScrollLock: true, children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: {
                    titleKey: "SelectDateRange",
                } }), _jsxs(Stack, { direction: "column", spacing: 2, m: 2, children: [_jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(TextField, { fullWidth: true, name: "start", label: t("Start"), type: "datetime-local", InputLabelProps: {
                                        shrink: true,
                                    }, value: after ? toDatetimeLocal(after) : "", onChange: (e) => handleAfterChange(fromDatetimeLocal(e.target.value)) }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(TextField, { fullWidth: true, name: "end", label: t("End"), type: "datetime-local", InputLabelProps: {
                                        shrink: true,
                                    }, value: before ? toDatetimeLocal(before) : "", onChange: (e) => handleBeforeChange(fromDatetimeLocal(e.target.value)) }) })] }), _jsx(Button, { type: "submit", fullWidth: true, onClick: () => { onSubmit(after, before); onClose(); }, children: t("Ok") })] })] }));
};
//# sourceMappingURL=DateRangeMenu.js.map