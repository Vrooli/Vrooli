import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CancelIcon, CreateIcon, SaveIcon } from "@local/icons";
import { exists } from "@local/utils";
import { Box, Button, CircularProgress, Grid } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useErrorPopover } from "../../../utils/hooks/useErrorPopover";
import { GridActionButtons } from "../GridActionButtons/GridActionButtons";
export const GridSubmitButtons = ({ disabledCancel, disabledSubmit, display, errors, isCreate, loading = false, onCancel, onSetSubmitting, onSubmit, }) => {
    const { t } = useTranslation();
    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting });
    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const isSubmitDisabled = useMemo(() => loading || hasErrors || (disabledSubmit === true), [disabledSubmit, hasErrors, loading]);
    const handleSubmit = useCallback((ev) => {
        if (hasErrors)
            openPopover(ev);
        else if (!disabledSubmit && typeof onSubmit === "function")
            onSubmit();
    }, [hasErrors, openPopover, disabledSubmit, onSubmit]);
    return (_jsxs(GridActionButtons, { display: display, children: [_jsx(Popover, {}), _jsx(Grid, { item: true, xs: 6, children: _jsx(Box, { onClick: handleSubmit, children: _jsx(Button, { disabled: isSubmitDisabled, fullWidth: true, startIcon: loading ? _jsx(CircularProgress, { size: 24, sx: { color: "white" } }) : (isCreate ? _jsx(CreateIcon, {}) : _jsx(SaveIcon, {})), children: t(isCreate ? "Create" : "Save") }) }) }), _jsx(Grid, { item: true, xs: 6, children: _jsx(Button, { disabled: loading || (disabledCancel !== undefined ? disabledCancel : false), fullWidth: true, onClick: onCancel, startIcon: _jsx(CancelIcon, {}), children: t("Cancel") }) })] }));
};
//# sourceMappingURL=GridSubmitButtons.js.map