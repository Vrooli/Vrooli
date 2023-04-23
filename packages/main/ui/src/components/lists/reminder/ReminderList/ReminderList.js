import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { DeleteType } from "@local/consts";
import { useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutationWrapper, useCustomMutation } from "../../../../api";
import { deleteOneOrManyDeleteOne } from "../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { ReminderUpsert } from "../../../../views/reminder";
import { TitleContainer } from "../../../containers/TitleContainer/TitleContainer";
import { LargeDialog } from "../../../dialogs/LargeDialog/LargeDialog";
import { ReminderListItem } from "../ReminderListItem/ReminderListItem";
export const ReminderList = ({ handleUpdate, listId, loading, reminders, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [allReminders, setAllReminders] = useState(reminders);
    useEffect(() => {
        setAllReminders(reminders);
    }, [reminders]);
    const [editingIndex, setEditingIndex] = useState(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);
    const handleCreated = useCallback((reminder) => {
        setAllReminders([...allReminders, reminder]);
        handleUpdate && handleUpdate([...allReminders, reminder]);
    }, [allReminders, handleUpdate]);
    const handleUpdated = useCallback((index, reminder) => {
        const newList = [...allReminders];
        newList[index] = reminder;
        setAllReminders(newList);
        handleUpdate && handleUpdate(newList);
    }, [allReminders, handleUpdate]);
    const handleCompleted = useCallback((reminder) => {
        if (editingIndex >= 0) {
            handleUpdated(editingIndex, reminder);
        }
        else {
            handleCreated(reminder);
        }
        closeDialog();
    }, []);
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback((index) => {
        const reminder = allReminders[index];
        mutationWrapper({
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
    return (_jsxs(_Fragment, { children: [_jsx(LargeDialog, { id: "reminder-dialog", onClose: closeDialog, isOpen: isDialogOpen, titleId: "", zIndex: zIndex + 1, children: _jsx(ReminderUpsert, { display: "dialog", partialData: editingIndex >= 0 ? reminders[editingIndex] : undefined, handleDelete: editingIndex >= 0 ? () => handleDelete(editingIndex) : () => { }, isCreate: editingIndex < 0, listId: listId ?? (editingIndex >= 0 ? reminders[editingIndex].reminderList.id : undefined), onCancel: closeDialog, onCompleted: handleCompleted, zIndex: zIndex + 1 }) }), _jsx(TitleContainer, { titleKey: "ToDo", options: [["Create", openDialog]], children: reminders.map((reminder, index) => (_jsx(ReminderListItem, { handleDelete: () => { }, handleOpen: () => openUpdateDialog(index), handleUpdate: (updated) => handleUpdated(index, updated), reminder: reminder, zIndex: zIndex }, `reminder-${index}`))) })] }));
};
//# sourceMappingURL=ReminderList.js.map