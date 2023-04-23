import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { DeleteIcon } from "@local/icons";
import { Button, DialogContent, Stack, TextField, Typography, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { deleteOneOrManyDeleteOne } from "../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { useLocation } from "../../../utils/route";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { LargeDialog } from "../LargeDialog/LargeDialog";
export const DeleteDialog = ({ handleClose, isOpen, objectId, objectName, objectType, zIndex, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [nameInput, setNameInput] = useState("");
    const close = useCallback((wasDeleted) => {
        setNameInput("");
        handleClose(wasDeleted ?? false);
    }, [handleClose]);
    const [deleteOne] = useCustomMutation(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback(() => {
        mutationWrapper({
            mutation: deleteOne,
            input: { id: objectId, objectType },
            successCondition: (data) => data.success,
            successMessage: () => ({ key: "ObjectDeleted", variables: { objectName } }),
            onSuccess: () => {
                setLocation(LINKS.Home);
                close(true);
            },
            errorMessage: () => ({ key: "FailedToDelete" }),
            onError: () => {
                close(false);
            },
        });
    }, [close, deleteOne, objectId, objectName, objectType, setLocation]);
    return (_jsxs(LargeDialog, { id: "delete-dialog", isOpen: isOpen, onClose: () => { close(); }, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: () => { close(); }, titleData: {
                    titleKey: "Delete",
                } }), _jsx(DialogContent, { children: _jsxs(Stack, { direction: "column", spacing: 2, mt: 2, children: [_jsxs(Typography, { variant: "h6", children: ["Are you absolutely certain you want to delete \"", objectName, "\"?"] }), _jsx(Typography, { variant: "body1", sx: { color: palette.background.textSecondary, paddingBottom: 3 }, children: "This action cannot be undone." }), _jsxs(Typography, { variant: "h6", children: ["Enter ", _jsx("b", { children: objectName }), " to confirm."] }), _jsx(TextField, { variant: "outlined", fullWidth: true, value: nameInput, onChange: (e) => setNameInput(e.target.value), error: nameInput.trim() !== objectName.trim(), helperText: nameInput.trim() !== objectName.trim() ? "Name does not match" : "", sx: { paddingBottom: 2 } }), _jsx(Button, { startIcon: _jsx(DeleteIcon, {}), color: "secondary", onClick: handleDelete, disabled: nameInput.trim() !== objectName.trim(), children: t("Delete") })] }) })] }));
};
//# sourceMappingURL=DeleteDialog.js.map