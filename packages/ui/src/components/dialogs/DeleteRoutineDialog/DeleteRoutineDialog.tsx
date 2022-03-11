import {
    Box,
    Button,
    Dialog,
    DialogContent,
    DialogTitle,
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
            <DialogTitle id="delete-routine-dialog-title">Delete {}</DialogTitle>
            <DialogContent>
                <Stack direction="column" spacing="2">
                    <Typography variant="h3">Are you absolutely certain you want to delete this routine?</Typography>
                    <Typography variant="body1" sx={{color: (t) => t.palette.background.textSecondary}}>This action cannot be undone. Subroutines will not be deleted.</Typography>
                    <Typography variant="h3">Please type <Box fontWeight='fontWeightMedium' display='inline'>{routineName}</Box> to confirm.</Typography>
                    <TextField 
                        label="Routine Name" 
                        variant="outlined" 
                        fullWidth 
                        value={routineNameInput} 
                        onChange={(e) => setRoutineNameInput(e.target.value)} 
                        error={routineNameInput !== routineName} 
                        helperText={routineNameInput !== routineName ? 'Routine name does not match' : ''} 
                    />
                    <Button color="secondary" onClick={handleDelete}>Delete</Button>
                </Stack>
            </DialogContent>
        </Dialog>
    )
}