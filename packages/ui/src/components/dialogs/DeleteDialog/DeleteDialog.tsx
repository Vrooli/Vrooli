import {
    Button,
    Dialog,
    DialogContent,
    Stack,
    TextField,
    Typography,
    useTheme
} from '@mui/material';
import { DeleteDialogProps } from '../types';
import { useCallback, useState } from 'react';
import { mutationWrapper } from 'graphql/utils';
import { useMutation } from '@apollo/client';
import { deleteOneMutation } from 'graphql/mutation';
import { deleteOne, deleteOneVariables } from 'graphql/generated/deleteOne';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { PubSub } from 'utils';
import { DialogTitle } from 'components';
import { DeleteIcon } from '@shared/icons';

const titleAria = 'delete-object-dialog-title';

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
            aria-labelledby={titleAria}
            sx={{
                zIndex
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title={`Delete "${objectName}"`}
                onClose={() => { close() }}
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
                        helperText={nameInput.trim() !== objectName.trim() ? 'Name does not match' : ''}
                        sx={{ paddingBottom: 2 }}
                    />
                    <Button startIcon={<DeleteIcon />} color="secondary" onClick={handleDelete}>Delete</Button>
                </Stack>
            </DialogContent>
        </Dialog>
    )
}