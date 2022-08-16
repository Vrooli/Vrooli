import {
    Box,
    Button,
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    TextField,
    Typography,
    useTheme
} from '@mui/material';
import { DeleteDialogProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';
import { useCallback, useState } from 'react';
import { mutationWrapper } from 'graphql/utils';
import { useMutation } from '@apollo/client';
import { deleteOneMutation } from 'graphql/mutation';
import { deleteOne, deleteOneVariables } from 'graphql/generated/deleteOne';
import { APP_LINKS } from '@local/shared';
import { useLocation } from '@local/route';
import { PubSub } from 'utils';

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

    // Stores user-inputted name of object to be deleted
    const [nameInput, setNameInput] = useState<string>('');

    const close = useCallback((wasDeleted?: boolean) => {
        setNameInput('');
        handleClose(wasDeleted ?? false);
    }, [handleClose]);

    const [deleteOne] = useMutation<deleteOne, deleteOneVariables>(deleteOneMutation);
    const handleDelete = useCallback(() => {
        mutationWrapper({
            mutation: deleteOne,
            input: { id: objectId, objectType },
            onSuccess: (response) => {
                if (response?.data?.deleteOne?.success) {
                    PubSub.get().publishSnack({ message: `${objectName} deleted.` });
                    setLocation(APP_LINKS.Home);
                } else {
                    PubSub.get().publishSnack({ message: `Error deleting ${objectName}.`, severity: 'error' });
                }
                close(true);
            },
            onError: () => {
                PubSub.get().publishSnack({ message: `Failed to delete ${objectName}.` });
                close(false);
            }
        })
    }, [close, deleteOne, objectId, objectName, objectType, setLocation]);

    return (
        <Dialog
            open={isOpen}
            onClose={() => { close(); }}
            aria-labelledby="delete-object-dialog-title"
            aria-describedby="delete-object-dialog-description"
            sx={{
                zIndex
            }}
        >
            <Box sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
            }}>
                <Typography variant="h6" sx={{ marginLeft: 'auto' }}>Delete {objectName}</Typography>
                <IconButton onClick={() => { close(); }} sx={{
                    justifyContent: 'end',
                    flexDirection: 'row-reverse',
                    marginLeft: 'auto',
                }}>
                    <CloseIcon fontSize="large" sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
            <DialogContent>
                <Stack direction="column" spacing="2">
                    <Typography variant="h6">Are you absolutely certain you want to delete "{objectName}"?</Typography>
                    <Typography variant="body1" sx={{ color: palette.background.textSecondary, paddingBottom: 3 }}>This action cannot be undone. Subroutines will not be deleted.</Typography>
                    <Typography variant="h6" sx={{ paddingBottom: 1 }}>Please type <b>{objectName}</b> to confirm.</Typography>
                    <TextField
                        label="Routine Name"
                        variant="outlined"
                        fullWidth
                        value={nameInput}
                        onChange={(e) => setNameInput(e.target.value)}
                        error={nameInput.trim() !== objectName.trim()}
                        helperText={nameInput.trim() !== objectName.trim() ? 'Name does not match' : ''}
                        sx={{ paddingBottom: 2 }}
                    />
                    <Button color="secondary" onClick={handleDelete}>Delete</Button>
                </Stack>
            </DialogContent>
        </Dialog>
    )
}