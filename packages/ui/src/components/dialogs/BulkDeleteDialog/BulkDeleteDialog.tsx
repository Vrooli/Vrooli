import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import Grid from "@mui/material/Grid";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { TextInput } from "../../inputs/TextInput/TextInput.js";
import { Dialog, DialogContent, DialogActions } from "../Dialog/Dialog.js";
import { type BulkDeleteDialogProps } from "../types.js";

// Delete confirmation prompt is 2 random bunny words
const BUNNY_WORDS = ["bunny", "rabbit", "boop", "binky", "zoom", "ears", "fluffy", "hop", "tail", "whiskers", "burrow", "nose", "grass", "meadow"];
function getRandomConfirmationWords() {
    const shuffled = BUNNY_WORDS.sort(() => 0.5 - Math.random());
    return `${shuffled[0]} ${shuffled[1]}`;
}

export function BulkDeleteDialog({
    handleClose,
    isOpen,
    selectedData,
}: BulkDeleteDialogProps) {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [selectedItems, setSelectedItems] = useState(selectedData);
    const [confirmationInput, setConfirmationInput] = useState("");
    const [randomWords, setRandomWords] = useState(getRandomConfirmationWords());
    useEffect(() => {
        if (isOpen) setRandomWords(getRandomConfirmationWords());
    }, [isOpen]);
    const isConfirmationValid = confirmationInput.toUpperCase() === randomWords.toUpperCase();

    function handleCheckboxChange(object) {
        if (selectedItems.includes(object)) {
            setSelectedItems(prev => prev.filter(item => item !== object));
        } else {
            setSelectedItems(prev => [...prev, object]);
        }
    }

    function toggleAllItems() {
        if (selectedItems.length === selectedData.length) {
            setSelectedItems([]);
        } else {
            setSelectedItems(selectedData);
        }
    }

    const onCancel = useCallback(() => {
        handleClose([]);
    }, [handleClose]);
    const onDelete = useCallback(() => {
        handleClose(selectedItems);
    }, [handleClose, selectedItems]);

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onCancel}
            title={t("DeleteBulkConfirm")}
            size="full"
            variant="danger"
        >
            <DialogContent>
                <Typography variant="body2" color="error" mb={2}>{t("DeleteBulkWarning")}</Typography>
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={selectedItems.length === selectedData.length}
                            indeterminate={selectedItems.length > 0 && selectedItems.length < selectedData.length}
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
                    {selectedData.map(object => (
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
                    Type the words &quot;{randomWords}&quot; to confirm deletion:
                </Typography>
                <TextInput
                    fullWidth
                    value={confirmationInput}
                    onChange={(e) => setConfirmationInput(e.target.value)}
                    sx={{ marginBottom: 2 }}
                />
            </DialogContent>
            <DialogActions variant="danger">
                <Button
                    fullWidth
                    startIcon={<IconCommon
                        decorative
                        name="Cancel"
                    />}
                    onClick={onCancel}
                    variant="outlined"
                >{t("Cancel")}</Button>
                <Button
                    disabled={!isConfirmationValid}
                    fullWidth
                    startIcon={<IconCommon
                        decorative
                        name="Delete"
                    />}
                    type="submit"
                    onClick={onDelete}
                    variant="contained"
                >{t("Delete")}</Button>
            </DialogActions>
        </Dialog>
    );
}
