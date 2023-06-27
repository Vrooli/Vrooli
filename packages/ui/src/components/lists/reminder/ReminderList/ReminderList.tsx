/**
 * Displays a list of emails for the user to manage
 */
import { AddIcon, DeleteOneInput, DeleteType, endpointPostDeleteOne, endpointPostReminder, endpointPutReminder, LINKS, OpenInNewIcon, Reminder, ReminderCreateInput, ReminderIcon, ReminderUpdateInput, Success, useLocation } from "@local/shared";
import { List, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TitleContainer } from "components/containers/TitleContainer/TitleContainer";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { MyStuffPageTabOption } from "utils/search/objectToSearch";
import { shapeReminder } from "utils/shape/models/reminder";
import { ReminderUpsert } from "views/objects/reminder";
import { ReminderListItem } from "../ReminderListItem/ReminderListItem";
import { ReminderListProps } from "../types";

export const ReminderList = ({
    handleUpdate,
    listId,
    loading,
    reminders,
    zIndex,
}: ReminderListProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

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
    }, [closeDialog, editingIndex, handleCreated, handleUpdated]);

    // Handle add mutation (for undo)
    const [addMutation, { errors: addErrors }] = useLazyFetch<ReminderCreateInput, Reminder>(endpointPostReminder);
    useDisplayServerError(addErrors);

    // Handle update mutation
    const [updateMutation, { errors: updateErrors }] = useLazyFetch<ReminderUpdateInput, Reminder>(endpointPutReminder);
    useDisplayServerError(updateErrors);
    const saveUpdate = useCallback((index: number, original: Reminder, updated: Reminder) => {
        // Don't wait for the mutation to call handleUpdated
        handleUpdated(index, updated);
        // Call the mutation
        fetchLazyWrapper<ReminderUpdateInput, Reminder>({
            fetch: updateMutation,
            inputs: shapeReminder.update(original, updated),
            successCondition: (data) => !!data.id,
            successMessage: () => ({ messageKey: "ObjectUpdated", messageVariables: { objectName: updated.name } }),
        });
    }, [updateMutation, handleUpdated]);

    // Handle delete
    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback((index: number) => {
        const reminder = allReminders[index];
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteMutation,
            inputs: { id: reminder.id, objectType: DeleteType.Reminder },
            successCondition: (data) => data.success,
            successMessage: () => ({
                messageKey: "ObjectDeleted",
                messageVariables: { objectName: reminder.name },
                buttonKey: "Undo",
                buttonClicked: () => {
                    fetchLazyWrapper<ReminderCreateInput, Reminder>({
                        fetch: addMutation,
                        inputs: shapeReminder.create({
                            ...reminder,
                            // Make sure not to set any extra fields, 
                            // so this is treated as a "Connect" instead of a "Create"
                            reminderList: {
                                __typename: "ReminderList" as const,
                                id: reminder.reminderList.id,
                            },
                        }),
                        successCondition: (data) => !!data.id,
                        onSuccess: (data) => { handleCreated(data); },
                    });
                },
            }),
            onSuccess: () => {
                const newList = [...allReminders];
                newList.splice(index, 1);
                setAllReminders(newList);
                handleUpdate && handleUpdate(newList);
                closeDialog();
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
        });
    }, [addMutation, allReminders, closeDialog, deleteMutation, handleCreated, handleUpdate]);

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
                Icon={ReminderIcon}
                title={t("ToDo")}
                options={[{
                    Icon: OpenInNewIcon,
                    label: t("SeeAll"),
                    onClick: () => { setLocation(`${LINKS.MyStuff}?type=${MyStuffPageTabOption.Reminders}`); },
                }, {
                    Icon: AddIcon,
                    label: t("Create"),
                    onClick: openDialog,
                }]}
            >
                <>
                    {/* Empty text */}
                    {reminders.length === 0 && <Typography variant="h6" sx={{
                        textAlign: "center",
                        paddingTop: "8px",
                    }}>{t("NoResults", { ns: "error" })}</Typography>}
                    {/* Existing reminders */}
                    <List>
                        {reminders.map((reminder, index) => (
                            <ReminderListItem
                                key={`reminder-${index}`}
                                handleDelete={() => handleDelete(index)}
                                handleOpen={() => openUpdateDialog(index)}
                                handleUpdate={(updated) => saveUpdate(index, reminder, updated)}
                                reminder={reminder}
                                zIndex={zIndex}
                            />
                        ))}
                    </List>
                </>
            </TitleContainer>
        </>
    );
};
