import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { AddIcon, CloseIcon, DeleteIcon, DragIcon, ListNumberIcon } from "@local/icons";
import { DUMMY_ID } from "@local/uuid";
import { reminderValidation } from "@local/validation";
import { Box, Button, IconButton, InputAdornment, Stack, TextField, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import { forwardRef, useCallback } from "react";
import { DragDropContext, Draggable, Droppable } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { Subheader } from "../../components/text/Subheader/Subheader";
import { getCurrentUser } from "../../utils/authentication/session";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeReminder } from "../../utils/shape/models/reminder";
import { BaseForm } from "../BaseForm/BaseForm";
export const reminderInitialValues = (session, reminderListId, existing) => ({
    __typename: "Reminder",
    id: DUMMY_ID,
    description: null,
    dueDate: null,
    index: 0,
    isComplete: false,
    name: "",
    reminderList: {
        __typename: "ReminderList",
        id: reminderListId ?? DUMMY_ID,
        ...(reminderListId === undefined && {
            focusMode: getCurrentUser(session)?.activeFocusMode?.mode,
        }),
    },
    reminderItems: [],
    ...existing,
});
export function transformReminderValues(values, existing) {
    return existing === undefined
        ? shapeReminder.create(values)
        : shapeReminder.update(existing, values);
}
export const validateReminderValues = async (values, existing) => {
    const transformedValues = transformReminderValues(values, existing);
    const validationSchema = reminderValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const ReminderForm = forwardRef(({ display, dirty, index, isCreate, isLoading, isOpen, onCancel, reminderListId, values, zIndex, ...props }, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, , dueDateHelpers] = useField("dueDate");
    const [reminderItemsField, , reminderItemsHelpers] = useField("reminderItems");
    const handleDeleteStep = (index) => {
        const newReminderItems = [...reminderItemsField.value];
        newReminderItems.splice(index, 1);
        reminderItemsHelpers.setValue(newReminderItems);
    };
    const handleAddStep = () => {
        reminderItemsHelpers.setValue([
            ...reminderItemsField.value,
            {
                id: Date.now(),
                name: "",
                description: "",
                dueDate: null,
            },
        ]);
    };
    const onDragEnd = (result) => {
        if (!result.destination)
            return;
        const newReminderItems = Array.from(reminderItemsField.value);
        const [reorderedItem] = newReminderItems.splice(result.source.index, 1);
        newReminderItems.splice(result.destination.index, 0, reorderedItem);
        reminderItemsHelpers.setValue(newReminderItems);
    };
    const clearDueDate = useCallback((index) => {
        if (index === undefined) {
            dueDateHelpers.setValue(null);
        }
        else {
            const newReminderItems = [...reminderItemsField.value];
            newReminderItems[index].dueDate = null;
            reminderItemsHelpers.setValue(newReminderItems);
        }
    }, [dueDateHelpers, reminderItemsField.value, reminderItemsHelpers]);
    return (_jsxs(_Fragment, { children: [_jsx(DragDropContext, { onDragEnd: onDragEnd, children: _jsx(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                        display: "block",
                        width: "min(500px, 100vw - 16px)",
                        margin: "auto",
                        paddingLeft: "env(safe-area-inset-left)",
                        paddingRight: "env(safe-area-inset-right)",
                        paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                    }, children: _jsxs(Stack, { direction: "column", spacing: 4, sx: {
                            margin: 2,
                            marginBottom: 4,
                        }, children: [_jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(Field, { fullWidth: true, name: "name", label: t("Name"), as: TextField }), _jsx(Field, { fullWidth: true, name: "description", label: t("Description"), as: TextField })] }), _jsx(Field, { fullWidth: true, name: "dueDate", label: "Due date (optional)", type: "datetime-local", InputProps: {
                                    endAdornment: (_jsxs(InputAdornment, { position: "end", sx: { display: "flex", alignItems: "center" }, children: [_jsx("input", { type: "hidden" }), _jsx(IconButton, { edge: "end", size: "small", onClick: () => { clearDueDate(); }, children: _jsx(CloseIcon, { fill: palette.background.textPrimary }) })] })),
                                }, InputLabelProps: {
                                    shrink: true,
                                }, as: TextField }), _jsx(Subheader, { Icon: ListNumberIcon, title: "Steps" }), _jsx(Droppable, { droppableId: "reminderItems", children: (provided) => (_jsxs("div", { ref: provided.innerRef, ...provided.droppableProps, children: [(reminderItemsField.value ?? []).map((reminderItem, i) => (_jsx(Draggable, { draggableId: String(reminderItem.id), index: i, children: (provided) => (_jsx(Box, { ref: provided.innerRef, ...provided.draggableProps, sx: {
                                                    borderRadius: 2,
                                                    border: `2px solid ${palette.divider}`,
                                                    marginBottom: 2,
                                                    padding: 1,
                                                }, children: _jsxs(Stack, { direction: "row", alignItems: "flex-start", spacing: 2, sx: { boxShadow: 6 }, children: [_jsxs(Stack, { spacing: 1, sx: { width: "100%" }, children: [_jsx(Field, { fullWidth: true, name: `reminderItems[${i}].name`, label: t("Name"), as: TextField }), _jsx(Field, { fullWidth: true, name: `reminderItems[${i}].description`, label: t("Description"), as: TextField }), _jsx(Field, { fullWidth: true, name: `reminderItems[${i}].dueDate`, label: "Due date (optional)", type: "datetime-local", InputProps: {
                                                                        endAdornment: (_jsxs(InputAdornment, { position: "end", sx: { display: "flex", alignItems: "center" }, children: [_jsx("input", { type: "hidden" }), _jsx(IconButton, { edge: "end", size: "small", onClick: () => { clearDueDate(i); }, children: _jsx(CloseIcon, { fill: palette.background.textPrimary }) })] })),
                                                                    }, InputLabelProps: {
                                                                        shrink: true,
                                                                    }, as: TextField })] }), _jsxs(Stack, { spacing: 1, width: 32, children: [_jsx(IconButton, { edge: "end", size: "small", onClick: () => handleDeleteStep(i), sx: { margin: "auto" }, children: _jsx(DeleteIcon, { fill: palette.error.light }) }), _jsx(Box, { ...provided.dragHandleProps, children: _jsx(DragIcon, { fill: palette.background.textPrimary }) })] })] }) })) }, reminderItem.id))), provided.placeholder] })) }), _jsx(Button, { startIcon: _jsx(AddIcon, {}), onClick: handleAddStep, sx: { alignSelf: "center", mt: 1 }, children: "Add Step" })] }) }) }), _jsx(GridSubmitButtons, { display: display, errors: props.errors, isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
});
//# sourceMappingURL=ReminderForm.js.map