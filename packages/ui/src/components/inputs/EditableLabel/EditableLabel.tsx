/**
 * Label that turns into a text input when clicked. 
 * Stores new text until committed.
 */
import { Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, IconButton, Stack, TextField, Typography, useTheme } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import {
    Cancel as CancelIcon,
    Close as CloseIcon,
    Done as DoneIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { EditableLabelProps } from '../types';
import { noSelect } from 'styles';

export const EditableLabel = ({
    canEdit,
    handleUpdate,
    placeholder,
    renderLabel,
    text,
    sxs,
}: EditableLabelProps) => {
    const { palette } = useTheme();

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
    const toggleActive = useCallback((event: any) => {
        if (!canEdit) return;
        setActive(!active)
    }, [active, canEdit]);
    const save = useCallback((event: React.MouseEvent<any>) => {
        handleUpdate(changedText);
        setActive(false);
    }, [changedText, handleUpdate]);
    const cancel = useCallback((event: React.MouseEvent<any>) => {
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
                aria-labelledby="edit-label-title"
                aria-describedby="edit-label-body"
                sx={{
                    '& .MuiPaper-root': {
                        minWidth: 'min(400px, 100%)',
                        margin: '0 auto',
                    },
                }}
            >
                {/* Title with close icon */}
                <DialogTitle
                    id="edit-label-title"
                    sx={{
                        ...noSelect,
                        display: 'flex',
                        alignItems: 'center',
                        padding: 2,
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                    }}
                >
                    <Typography
                        variant="h6"
                        sx={{
                            width: '-webkit-fill-available',
                            textAlign: 'center',
                        }}
                    >
                        Edit Label
                    </Typography>
                    <IconButton
                        aria-label="close"
                        edge="start"
                        onClick={cancel}
                    >
                        <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                    </IconButton>
                </DialogTitle>
                <DialogContent>
                    <DialogContentText id="edit-label-body">
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
                        onClick={save}
                        color="secondary"
                        startIcon={<DoneIcon />}
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