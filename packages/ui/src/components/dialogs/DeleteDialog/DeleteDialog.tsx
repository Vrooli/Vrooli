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
import { mutationWrapper } from 'api/utils';
import { useCustomMutation } from 'api/hooks';
import { APP_LINKS, DeleteOneInput, Success } from '@shared/consts';
import { useLocation } from '@shared/route';
import { DialogTitle } from 'components';
import { DeleteIcon } from '@shared/icons';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany_deleteOne';

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

    const [deleteOne] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback(() => {
        mutationWrapper<Success, DeleteOneInput>({
            mutation: deleteOne,
            input: { id: objectId, objectType },
            successCondition: (data) => data.success,
            successMessage: () => ({ key: 'ObjectDeleted', variables: { objectName } }),
            onSuccess: () => {
                setLocation(APP_LINKS.Home);
                close(true);
            },
            errorMessage: () => ({ key: 'FailedToDelete' }),
            onError: () => {
                close(false);
            }
        })
    }, [close, deleteOne, objectId, objectName, objectType, setLocation]);

    return (
        <Dialog
            open={isOpen}
            onClose={() => { close(); }}
            sx={{
                zIndex
            }}
        >
            <DialogTitle
                ariaLabel=''
                title=''
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