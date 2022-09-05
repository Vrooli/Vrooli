/**
 * Label that turns into a text input when clicked. 
 * Stores new text until committed.
 */
import { Button, Dialog, DialogActions, DialogContent, DialogContentText, IconButton, Stack, TextField } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { EditableLabelProps } from '../types';
import { DialogTitle } from 'components/dialogs';
import { CancelIcon, EditIcon, SaveIcon } from '@shared/icons';

const titleAria = 'editable-label-dialog-title';
const descriptionAria = 'editable-label-dialog-description';

export const EditableLabel = ({
    canEdit,
    handleUpdate,
    placeholder,
    onDialogOpen,
    renderLabel,
    text,
    sxs,
}: EditableLabelProps) => {

    /**
     * Random string for unique ID
     */
    const [id] = useState(Math.random().toString(36).substr(2, 9));

    // Stores changed text before committing
    const [changedText, setChangedText] = useState<string>(text);
    useEffect(() => {
        setChangedText(text);
    }, [text]);
    const onTextChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setChangedText(e.target.value);
    }, []);

    // Used for editing the title of the routine
    const [active, setActive] = useState<boolean>(false);
    useEffect(() => {
        if (typeof onDialogOpen === 'function') {
            onDialogOpen(active);
        }
    }, [active, onDialogOpen]);
    const toggleActive = useCallback((event: any) => {
        if (!canEdit) return;
        setActive(!active)
    }, [active, canEdit]);
    const save = useCallback(() => {
        handleUpdate(changedText);
        setActive(false);
    }, [changedText, handleUpdate]);
    const cancel = useCallback(() => {
        setChangedText(text ?? '');
        setActive(false);
    }, [text]);

    return (
        <>
            {/* Dialog with TextField for editing label */}
            <Dialog
                open={active}
                disableScrollLock={true}
                onClose={() => { }}
                aria-labelledby={titleAria}
                aria-describedby={descriptionAria}
                sx={{
                    '& .MuiPaper-root': {
                        minWidth: 'min(400px, 100%)',
                        margin: '0 auto',
                    },
                }}
            >
                <DialogTitle
                    ariaLabel={titleAria}
                    onClose={cancel}
                    title="Edit Label"
                />
                <DialogContent>
                    <DialogContentText id={descriptionAria}>
                        <TextField
                            autoFocus
                            margin="dense"
                            id="edit-label-input"
                            label="Label"
                            type="text"
                            fullWidth
                            value={changedText}
                            onChange={onTextChange}
                        />
                    </DialogContentText>
                </DialogContent>
                {/* Save and cancel buttons */}
                <DialogActions>
                    <Button
                        type="submit"
                        onClick={save}
                        color="secondary"
                        startIcon={<SaveIcon />}
                    >
                        Save
                    </Button>
                    <Button
                        onClick={cancel}
                        color="secondary"
                        startIcon={<CancelIcon />}
                    >
                        Cancel
                    </Button>
                </DialogActions>
            </Dialog >
            {/* Non-popup elements */}
            <Stack direction="row" spacing={0} alignItems="center" sx={{ ...(sxs?.stack ?? {}) }}>
                {/* Label */}
                {renderLabel(text.trim().length > 0 ? text : (placeholder ?? ''))}
                {/* Edit icon */}
                {canEdit && (
                    <IconButton
                        id={`edit-label-icon-button-${id}`}
                        onClick={toggleActive}
                        onTouchStart={toggleActive}
                        sx={{ color: 'inherit' }}
                    >
                        <EditIcon id={`edit-label-icon-${id}`} />
                    </IconButton>
                )}
            </Stack>
        </>
    )
};