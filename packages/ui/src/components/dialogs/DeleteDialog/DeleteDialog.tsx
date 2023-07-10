import { DeleteIcon, DeleteOneInput, endpointPostDeleteOne, LINKS, Success, useLocation } from "@local/shared";
import { Button, DialogContent, Stack, TextField, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { DeleteDialogProps } from "../types";

export const DeleteDialog = ({
    handleClose,
    isOpen,
    objectId,
    objectName,
    objectType,
    zIndex,
}: DeleteDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Stores user-inputted name of object to be deleted
    const [nameInput, setNameInput] = useState<string>("");

    const close = useCallback((wasDeleted?: boolean) => {
        setNameInput("");
        handleClose(wasDeleted ?? false);
    }, [handleClose]);

    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback(() => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id: objectId, objectType },
            successCondition: (data) => data.success,
            successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { objectName } }),
            onSuccess: () => {
                setLocation(LINKS.Home);
                close(true);
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
            onError: () => {
                close(false);
            },
        });
    }, [close, deleteOne, objectId, objectName, objectType, setLocation]);

    return (
        <LargeDialog
            id="delete-dialog"
            isOpen={isOpen}
            onClose={() => { close(); }}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={() => { close(); }}
                title={t("Delete")}
                zIndex={zIndex + 1000}
            />
            <DialogContent>
                <Stack direction="column" spacing={2} mt={2}>
                    <Typography variant="h6">Are you absolutely certain you want to delete "{objectName}"?</Typography>
                    <Typography variant="body1" sx={{ color: palette.background.textSecondary, paddingBottom: 3 }}>This action cannot be undone.</Typography>
                    <Typography variant="h6">Enter <b>{objectName}</b> to confirm.</Typography>
                    <TextField
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
};
