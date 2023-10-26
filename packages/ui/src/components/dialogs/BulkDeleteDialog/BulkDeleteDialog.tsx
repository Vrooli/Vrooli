import { Button, Checkbox, DialogContent, FormControlLabel, Grid, List, ListItem, TextField, Typography, useTheme } from "@mui/material";
import { BottomActionsGrid } from "components/buttons/BottomActionsGrid/BottomActionsGrid";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { CancelIcon, DeleteIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { BulkDeleteDialogProps } from "../types";

// Delete confirmation prompt is 2 random bunny words
const BUNNY_WORDS = ["bunny", "rabbit", "boop", "binky", "zoom", "ears", "fluffy", "hop", "tail", "whiskers", "burrow", "nose", "grass", "meadow"];
function getRandomConfirmationWords() {
    const shuffled = BUNNY_WORDS.sort(() => 0.5 - Math.random());
    return `${shuffled[0]} ${shuffled[1]}`;
}

export const BulkDeleteDialog = ({
    handleClose,
    isOpen,
    objects,
}: BulkDeleteDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [selectedItems, setSelectedItems] = useState(objects);
    const [confirmationInput, setConfirmationInput] = useState("");
    const [randomWords, setRandomWords] = useState(getRandomConfirmationWords());
    useEffect(() => {
        if (isOpen) setRandomWords(getRandomConfirmationWords());
    }, [isOpen]);
    const isConfirmationValid = confirmationInput.toUpperCase() === randomWords.toUpperCase();

    const handleCheckboxChange = (object) => {
        if (selectedItems.includes(object)) {
            setSelectedItems(prev => prev.filter(item => item !== object));
        } else {
            setSelectedItems(prev => [...prev, object]);
        }
    };

    const toggleAllItems = () => {
        if (selectedItems.length === objects.length) {
            setSelectedItems([]);
        } else {
            setSelectedItems(objects);
        }
    };

    const onCancel = useCallback(() => {
        handleClose([]);
    }, [handleClose]);
    const onDelete = useCallback(() => {
        handleClose(selectedItems);
    }, [handleClose, selectedItems]);

    return (
        <LargeDialog
            id="delete-dialog"
            isOpen={isOpen}
            onClose={onCancel}
        >
            <TopBar
                display="dialog"
                onClose={onCancel}
                title={t("DeleteBulkConfirm")}
            />
            <DialogContent>
                <Typography variant="body2" color="error" mb={2}>{t("DeleteBulkWarning")}</Typography>
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={selectedItems.length === objects.length}
                            indeterminate={selectedItems.length > 0 && selectedItems.length < objects.length}
                            onChange={toggleAllItems}
                        />
                    }
                    label={t("ToggleAll")}
                    sx={{ marginBottom: 1 }}
                />
                <List sx={{
                    background: palette.background.paper,
                    borderRadius: 1,
                    marginBottom: 2,
                }}>
                    {objects.map(object => (
                        <ListItem key={object.id}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={selectedItems.includes(object)}
                                        onChange={() => handleCheckboxChange(object)}
                                    />
                                }
                                label={getDisplay(object).title}
                            />
                        </ListItem>
                    ))}
                </List>
                <Typography variant="body2">
                    Type the words "{randomWords}" to confirm deletion:
                </Typography>
                <TextField
                    fullWidth
                    value={confirmationInput}
                    onChange={(e) => setConfirmationInput(e.target.value)}
                    sx={{marginBottom: 2}}
                />
            </DialogContent>
            <BottomActionsGrid display="dialog">
                <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                    <Button
                        disabled={!isConfirmationValid}
                        fullWidth
                        startIcon={<DeleteIcon />}
                        type="submit"
                        onClick={onDelete}
                        variant="contained"
                    >{t("Delete")}</Button>
                </Grid>
                <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                    <Button
                        fullWidth
                        startIcon={<CancelIcon />}
                        onClick={onCancel}
                        variant="outlined"
                    >{t("Cancel")}</Button>
                </Grid>
            </BottomActionsGrid>
        </LargeDialog>
    );
};
