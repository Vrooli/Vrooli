import {
    Box,
    Button,
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    TextField,
    Typography
} from '@mui/material';
import { DeleteRoutineDialogProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';
import { useState } from 'react';

export const DeleteRoutineDialog = ({
    handleClose,
    handleDelete,
    isOpen,
    routineName,
}: DeleteRoutineDialogProps) => {
    // Stores user-inputted name of routine to be deleted
    const [routineNameInput, setRoutineNameInput] = useState<string>(routineName);

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            aria-labelledby="delete-routine-dialog-title"
            aria-describedby="delete-routine-dialog-description"
        >
            <Box sx={{
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
            }}>
                <Typography variant="h6" sx={{ marginLeft: 'auto' }}>Delete {routineName}</Typography>
                <IconButton onClick={handleClose} sx={{
                    justifyContent: 'end',
                    flexDirection: 'row-reverse',
                    marginLeft: 'auto',
                }}>
                    <CloseIcon fontSize="large" sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Box>
            <DialogContent>
                <Stack direction="column" spacing="2">
                    <Typography variant="h6">Are you absolutely certain you want to delete this routine?</Typography>
                    <Typography variant="body1" sx={{ color: (t) => t.palette.background.textSecondary, paddingBottom: 3 }}>This action cannot be undone. Subroutines will not be deleted.</Typography>
                    <Typography variant="h6" sx={{ paddingBottom: 1 }}>Please type <b>{routineName}</b> to confirm.</Typography>
                    <TextField
                        label="Routine Name"
                        variant="outlined"
                        fullWidth
                        value={routineNameInput}
                        onChange={(e) => setRoutineNameInput(e.target.value)}
                        error={routineNameInput !== routineName}
                        helperText={routineNameInput !== routineName ? 'Routine name does not match' : ''}
                        sx={{ paddingBottom: 2 }}
                    />
                    <Button color="secondary" onClick={handleDelete}>Delete</Button>
                </Stack>
            </DialogContent>
        </Dialog>
    )
}