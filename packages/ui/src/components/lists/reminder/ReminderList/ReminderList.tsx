/**
 * Displays a list of emails for the user to manage
 */
import { useTheme } from '@mui/material';
import { DeleteOneInput, Reminder, Success } from '@shared/consts';
import { useCustomMutation } from 'api';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany_deleteOne';
import { ListContainer } from 'components/containers/ListContainer/ListContainer';
import { TitleContainer } from 'components/containers/TitleContainer/TitleContainer';
import { ReminderDialog } from 'components/dialogs/ReminderDialog/ReminderDialog';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ReminderListItem } from '../ReminderListItem/ReminderListItem';
import { ReminderListProps } from '../types';

export const ReminderList = ({
    handleUpdate,
    listId,
    loading,
    reminders,
    zIndex,
}: ReminderListProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Internal state
    const [allReminders, setAllReminders] = useState<Reminder[]>(reminders);
    useEffect(() => {
        setAllReminders(reminders);
    }, [reminders]);
    const handleCreated = useCallback((reminder: Reminder) => {
        setAllReminders([...allReminders, reminder]);
        handleUpdate && handleUpdate([...allReminders, reminder]);
    }, [allReminders, handleUpdate]);
    const handleUpdated = useCallback((index: number, reminder: Reminder) => {
        const newList = [...allReminders];
        newList[index] = reminder;
        setAllReminders(newList);
        handleUpdate && handleUpdate(newList);
    }, [allReminders, handleUpdate]);

    // Handle delete
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    // const onDelete = useCallback((email: Email) => {
    //     if (loadingDelete) return;
    //     // Make sure that the user has at least one other authentication method 
    //     // (i.e. one other email or one other wallet)
    //     if (list.length <= 1 && numVerifiedWallets === 0) {
    //         PubSub.get().publishSnack({ messageKey: 'MustLeaveVerificationMethod', severity: 'Error' });
    //         return;
    //     }
    //     // Confirmation dialog
    //     PubSub.get().publishAlertDialog({
    //         messageKey: 'EmailDeleteConfirm',
    //         messageVariables: { emailAddress: email.emailAddress },
    //         buttons: [
    //             {
    //                 labelKey: 'Yes',
    //                 onClick: () => {
    //                     mutationWrapper<Success, DeleteOneInput>({
    //                         mutation: deleteMutation,
    //                         input: { id: email.id, objectType: DeleteType.Email },
    //                         onSuccess: () => {
    //                             handleUpdate([...list.filter(w => w.id !== email.id)])
    //                         },
    //                     })
    //                 }
    //             },
    //             { labelKey: 'Cancel', onClick: () => { } },
    //         ]
    //     });
    // }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedWallets]);

    // Add/update resource dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true) }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1) }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true)
    }, []);

    return (
        <>
            {/* Dialog */}
            <ReminderDialog
                partialData={editingIndex >= 0 ? reminders[editingIndex as number] : undefined}
                index={editingIndex}
                isOpen={isDialogOpen}
                listId={listId ?? (editingIndex >= 0 ? reminders[editingIndex as number].reminderList.id : undefined) ?? ''}
                onClose={closeDialog}
                onCreated={handleCreated}
                onUpdated={handleUpdated}
                zIndex={zIndex + 1}
            />
            {/* List */}
            <TitleContainer
                titleKey="ToDo"
                options={[['Create', openDialog]]}
            >
                <ListContainer
                    isEmpty={reminders.length === 0}
                    sx={{ maxWidth: '500px' }}
                >
                    {/* Existing reminders */}
                    {reminders.map((reminder, index) => (
                        <ReminderListItem
                            key={`reminder-${index}`}
                            handleDelete={() => { }}
                            handleUpdate={(updated) => handleUpdated(index, updated)}
                            reminder={reminder}
                            zIndex={zIndex}
                        />
                    ))}
                </ListContainer>
            </TitleContainer>
        </>
    )
}