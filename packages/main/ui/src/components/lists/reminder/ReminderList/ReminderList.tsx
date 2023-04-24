/**
 * Displays a list of emails for the user to manage
 */
import { DeleteOneInput, DeleteType, Reminder, Success } from "@local/consts";
import { useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutationWrapper, useCustomMutation } from "../../../../api";
import { deleteOneOrManyDeleteOne } from "../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { ReminderUpsert } from "../../../../views/reminder";
import { TitleContainer } from "../../../containers/TitleContainer/TitleContainer";
import { LargeDialog } from "../../../dialogs/LargeDialog/LargeDialog";
import { ReminderListItem } from "../ReminderListItem/ReminderListItem";
import { ReminderListProps } from "../types";

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

    // Add/update resource dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);

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
    const handleCompleted = useCallback((reminder: Reminder) => {
        if (editingIndex >= 0) {
            handleUpdated(editingIndex, reminder);
        } else {
            handleCreated(reminder);
        }
        closeDialog();
    }, []);

    // Handle delete
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback((index: number) => {
        const reminder = allReminders[index];
        mutationWrapper<Success, DeleteOneInput>({
            mutation: deleteMutation,
            input: { id: reminder.id, objectType: DeleteType.Reminder },
            successCondition: (data) => data.success,
            successMessage: () => ({ key: "ObjectDeleted", variables: { objectName: reminder.name } }),
            onSuccess: () => {
                const newList = [...allReminders];
                newList.splice(index, 1);
                setAllReminders(newList);
                handleUpdate && handleUpdate(newList);
                closeDialog();
            },
            errorMessage: () => ({ key: "FailedToDelete" }),
        });
    }, [allReminders, deleteMutation, handleUpdate, loadingDelete]);

    return (
        <>
            {/* Dialog */}
            <LargeDialog
                id="reminder-dialog"
                onClose={closeDialog}
                isOpen={isDialogOpen}
                titleId={""}
                zIndex={zIndex + 1}
            >
                <ReminderUpsert
                    display="dialog"
                    partialData={editingIndex >= 0 ? reminders[editingIndex as number] : undefined}
                    handleDelete={editingIndex >= 0 ? () => handleDelete(editingIndex as number) : () => { }}
                    isCreate={editingIndex < 0}
                    listId={listId ?? (editingIndex >= 0 ? reminders[editingIndex as number].reminderList.id : undefined)}
                    onCancel={closeDialog}
                    onCompleted={handleCompleted}
                    zIndex={zIndex + 1}
                />
            </LargeDialog>
            {/* List */}
            <TitleContainer
                titleKey="ToDo"
                options={[["Create", openDialog]]}
            >
                {/* Existing reminders */}
                {reminders.map((reminder, index) => (
                    <ReminderListItem
                        key={`reminder-${index}`}
                        handleDelete={() => { }}
                        handleOpen={() => openUpdateDialog(index)}
                        handleUpdate={(updated) => handleUpdated(index, updated)}
                        reminder={reminder}
                        zIndex={zIndex}
                    />
                ))}
            </TitleContainer>
        </>
    );
};