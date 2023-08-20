/**
 * Displays a list of emails for the user to manage
 */
import { endpointPostReminder, endpointPutReminder, GqlModelType, LINKS, Reminder, ReminderCreateInput, ReminderUpdateInput } from "@local/shared";
import { List, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TitleContainer } from "components/containers/TitleContainer/TitleContainer";
import { useDisplayServerError } from "hooks/useDisplayServerError";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, OpenInNewIcon, ReminderIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { MyStuffPageTabOption } from "utils/search/objectToSearch";
import { shapeReminder } from "utils/shape/models/reminder";
import { ReminderCrud } from "views/objects/reminder";
import { ReminderListItem } from "../ReminderListItem/ReminderListItem";
import { ReminderListProps } from "../types";

export const ReminderList = ({
    handleUpdate,
    id,
    listId,
    loading,
    reminders,
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
    const openUpdateDialog = useCallback((data: Reminder) => {
        const index = allReminders.findIndex((reminder) => reminder.id === data.id);
        if (index < 0) return;
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, [allReminders]);

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
    const saveUpdate = useCallback((updated: Reminder) => {
        const index = allReminders.findIndex((reminder) => reminder.id === updated.id);
        if (index < 0) return;
        const original = allReminders[index];
        // Don't wait for the mutation to call handleUpdated
        handleUpdated(index, updated);
        // Call the mutation
        fetchLazyWrapper<ReminderUpdateInput, Reminder>({
            fetch: updateMutation,
            inputs: shapeReminder.update(original, updated),
            successCondition: (data) => !!data.id,
            successMessage: () => ({ messageKey: "ObjectUpdated", messageVariables: { objectName: updated.name } }),
        });
    }, [allReminders, handleUpdated, updateMutation]);

    const handleDeleted = useCallback((id: string) => {
        const index = allReminders.findIndex((reminder) => reminder.id === id);
        if (index < 0) return;
        const newList = [...allReminders];
        newList.splice(index, 1);
        setAllReminders(newList);
        handleUpdate && handleUpdate(newList);
        closeDialog();
    }, [allReminders, closeDialog, handleUpdate]);

    const onAction = useCallback((action: "Delete" | "Update", data: any) => {
        switch (action) {
            // case "Delete": //TODO
            //     handleDelete(data);
            //     break;
            case "Update":
                saveUpdate(data);
                break;
        }
    }, [saveUpdate]);

    return (
        <>
            {/* Dialog */}
            <ReminderCrud
                isCreate={editingIndex < 0}
                isOpen={isDialogOpen}
                onCancel={closeDialog}
                onCompleted={handleCompleted}
                onDeleted={handleDeleted}
                overrideObject={editingIndex >= 0 ? reminders[editingIndex as number] : { __typename: "Reminder", reminderList: { id: listId ?? "" } }}
            />
            {/* List */}
            <TitleContainer
                Icon={ReminderIcon}
                id={id}
                title={t("ToDo")}
                options={[{
                    Icon: OpenInNewIcon,
                    label: t("SeeAll"),
                    onClick: () => { setLocation(`${LINKS.MyStuff}?type=${MyStuffPageTabOption.Reminder}`); },
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
                                data={reminder}
                                loading={false}
                                objectType={GqlModelType.Reminder}
                                onAction={onAction}
                                onClick={openUpdateDialog}
                            />
                        ))}
                    </List>
                </>
            </TitleContainer>
        </>
    );
};
