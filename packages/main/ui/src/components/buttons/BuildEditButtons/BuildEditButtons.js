import { jsx as _jsx } from "react/jsx-runtime";
import { Box, Grid } from "@mui/material";
import { GridSubmitButtons } from "../GridSubmitButtons/GridSubmitButtons";
export const BuildEditButtons = ({ canSubmitMutate, canCancelMutate, errors, handleCancel, handleSubmit, isAdding, isEditing, loading, }) => {
    if (!isEditing)
        return null;
    return (_jsx(Box, { sx: {
            alignItems: "center",
            background: "transparent",
            display: "flex",
            justifyContent: "center",
            position: "absolute",
            zIndex: 2,
            bottom: 0,
            right: 0,
            paddingBottom: "calc(16px + env(safe-area-inset-bottom))",
            paddingLeft: "calc(16px + env(safe-area-inset-left))",
            paddingRight: "calc(16px + env(safe-area-inset-right))",
            height: "calc(64px + env(safe-area-inset-bottom))",
        }, children: _jsx(Grid, { container: true, spacing: 1, sx: { width: "min(100%, 600px)" }, children: _jsx(GridSubmitButtons, { disabledCancel: loading || !canCancelMutate, disabledSubmit: loading || !canSubmitMutate, display: "page", errors: errors, isCreate: isAdding, onCancel: handleCancel, onSubmit: handleSubmit }) }) }));
};
//# sourceMappingURL=BuildEditButtons.js.map