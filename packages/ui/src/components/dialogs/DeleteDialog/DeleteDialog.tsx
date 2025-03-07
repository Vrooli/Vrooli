import { Button, DialogContent, Stack, Typography, useTheme } from "@mui/material";
import { TextInput } from "components/inputs/TextInput/TextInput.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { DeleteIcon } from "icons/common.js";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";
import { DeleteDialogProps } from "../types.js";

export function DeleteDialog({
    handleClose,
    handleDelete,
    isOpen,
    objectName,
}: DeleteDialogProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Stores user-inputted name of object to be deleted
    const [nameInput, setNameInput] = useState<string>("");
    useEffect(() => {
        setNameInput("");
    }, [isOpen]);

    const close = useCallback((wasDeleted?: boolean) => {
        handleClose(wasDeleted ?? false);
    }, [handleClose]);

    function onClose() {
        close();
    }

    return (
        <LargeDialog
            id="delete-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("Delete")}
            />
            <DialogContent>
                <Stack direction="column" spacing={2} mt={2}>
                    <Typography variant="h6">{t("DeleteConfirmTitle", { objectName })}</Typography>
                    <Typography variant="body1" sx={{ color: palette.background.textSecondary, paddingBottom: 3 }}>{t("DeleteConfirmDescription")}</Typography>
                    <Typography variant="h6">Enter <b>{objectName}</b> to confirm.</Typography>
                    <TextInput
                        variant="outlined"
                        fullWidth
                        value={nameInput}
                        onChange={(e) => setNameInput(e.target.value)}
                        error={nameInput.trim() !== objectName.trim()}
                        helperText={nameInput.trim() !== objectName.trim() ? "Name does not match" : ""}
                        sx={{ paddingBottom: 2 }}
                    />
                    <Button
                        startIcon={<DeleteIcon />}
                        color="secondary"
                        onClick={handleDelete}
                        disabled={nameInput.trim() !== objectName.trim()}
                        variant="contained"
                    >{t("Delete")}</Button>
                </Stack>
            </DialogContent>
        </LargeDialog>
    );
}
