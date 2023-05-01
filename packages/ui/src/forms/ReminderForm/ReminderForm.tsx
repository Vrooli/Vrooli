import { AddIcon, CloseIcon, DeleteIcon, DragIcon, DUMMY_ID, ListNumberIcon, Reminder, reminderValidation, Session } from "@local/shared";
import { Box, Button, IconButton, InputAdornment, Stack, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { Subheader } from "components/text/Subheader/Subheader";
import { Field, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ReminderFormProps } from "forms/types";
import { forwardRef, useCallback } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ReminderShape, shapeReminder } from "utils/shape/models/reminder";

export const reminderInitialValues = (
    session: Session | undefined,
    reminderListId: string | undefined,
    existing?: Reminder | null | undefined,
): ReminderShape => ({
    __typename: "Reminder" as const,
    id: DUMMY_ID,
    description: null,
    dueDate: null,
    index: 0,
    isComplete: false,
    name: "",
    reminderList: {
        __typename: "ReminderList" as const,
        id: reminderListId ?? DUMMY_ID,
        // If there's no reminderListId, add additional fields to create a new reminderList
        ...(reminderListId === undefined && {
            focusMode: getCurrentUser(session)?.activeFocusMode?.mode,
        }),
    },
    reminderItems: [],
    ...existing,
});

export function transformReminderValues(values: ReminderShape, existing?: ReminderShape) {
    return existing === undefined
        ? shapeReminder.create(values)
        : shapeReminder.update(existing, values);
}

export const validateReminderValues = async (values: ReminderShape, existing?: ReminderShape) => {
    const transformedValues = transformReminderValues(values, existing);
    const validationSchema = reminderValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ReminderForm = forwardRef<any, ReminderFormProps>(({
    display,
    dirty,
    index,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    reminderListId,
    values,
    zIndex,
    ...props
}, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [, , dueDateHelpers] = useField("dueDate");
    const [reminderItemsField, , reminderItemsHelpers] = useField("reminderItems");

    const handleDeleteStep = (index: number) => {
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

    const onDragEnd = (result: DropResult) => {
        if (!result.destination) return;

        const newReminderItems = Array.from(reminderItemsField.value);
        const [reorderedItem] = newReminderItems.splice(result.source.index, 1);
        newReminderItems.splice(result.destination.index, 0, reorderedItem);

        reminderItemsHelpers.setValue(newReminderItems);
    };

    const clearDueDate = useCallback((index?: number) => {
        // If no index is provided, clear the due date for the reminder
        if (index === undefined) {
            dueDateHelpers.setValue(null);
        }
        // Otherwise, we're clearing the due date for a specific reminder item (step)
        else {
            const newReminderItems = [...reminderItemsField.value];
            newReminderItems[index].dueDate = null;
            reminderItemsHelpers.setValue(newReminderItems);
        }
    }, [dueDateHelpers, reminderItemsField.value, reminderItemsHelpers]);

    return (
        <>
            <DragDropContext onDragEnd={onDragEnd}>
                <BaseForm
                    dirty={dirty}
                    isLoading={isLoading}
                    ref={ref}
                    style={{
                        display: "block",
                        width: "min(500px, 100vw - 16px)",
                        margin: "auto",
                        paddingLeft: "env(safe-area-inset-left)",
                        paddingRight: "env(safe-area-inset-right)",
                        paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                    }}
                >
                    <Stack direction="column" spacing={4} sx={{
                        margin: 2,
                        marginBottom: 4,
                    }}>
                        <Stack direction="column" spacing={2}>
                            <Field
                                fullWidth
                                name="name"
                                label={t("Name")}
                                as={TextField}
                            />
                            <Field
                                fullWidth
                                name="description"
                                label={t("Description")}
                                as={TextField}
                            />
                        </Stack>
                        <Field
                            fullWidth
                            name="dueDate"
                            label={"Due date (optional)"}
                            type="datetime-local"
                            InputProps={{
                                endAdornment: (
                                    <InputAdornment position="end" sx={{ display: "flex", alignItems: "center" }}>
                                        <input type="hidden" />
                                        <IconButton edge="end" size="small" onClick={() => { clearDueDate(); }}>
                                            <CloseIcon fill={palette.background.textPrimary} />
                                        </IconButton>
                                    </InputAdornment>
                                ),
                            }}
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        {/* Steps to complete reminder */}
                        <Subheader
                            Icon={ListNumberIcon}
                            title="Steps"
                        />
                        <Droppable droppableId="reminderItems">
                            {(provided) => (
                                <div ref={provided.innerRef} {...provided.droppableProps}>
                                    {
                                        (reminderItemsField.value ?? []).map((reminderItem, i) => (
                                            <Draggable key={reminderItem.id} draggableId={String(reminderItem.id)} index={i}>
                                                {(provided) => (
                                                    <Box
                                                        ref={provided.innerRef}
                                                        {...provided.draggableProps}
                                                        sx={{
                                                            borderRadius: 2,
                                                            border: `2px solid ${palette.divider}`,
                                                            marginBottom: 2,
                                                            padding: 1,
                                                        }}>
                                                        <Stack
                                                            direction="row"
                                                            alignItems="flex-start"
                                                            spacing={2}
                                                            sx={{ boxShadow: 6 }}
                                                        >
                                                            <Stack spacing={1} sx={{ width: "100%" }}>
                                                                <Field
                                                                    fullWidth
                                                                    name={`reminderItems[${i}].name`}
                                                                    label={t("Name")}
                                                                    as={TextField}
                                                                />
                                                                <Field
                                                                    fullWidth
                                                                    name={`reminderItems[${i}].description`}
                                                                    label={t("Description")}
                                                                    as={TextField}
                                                                />
                                                                <Field
                                                                    fullWidth
                                                                    name={`reminderItems[${i}].dueDate`}
                                                                    label={"Due date (optional)"}
                                                                    type="datetime-local"
                                                                    InputProps={{
                                                                        endAdornment: (
                                                                            <InputAdornment position="end" sx={{ display: "flex", alignItems: "center" }}>
                                                                                <input type="hidden" />
                                                                                <IconButton edge="end" size="small" onClick={() => { clearDueDate(i); }}>
                                                                                    <CloseIcon fill={palette.background.textPrimary} />
                                                                                </IconButton>
                                                                            </InputAdornment>
                                                                        ),
                                                                    }}
                                                                    InputLabelProps={{
                                                                        shrink: true,
                                                                    }}
                                                                    as={TextField}
                                                                />
                                                            </Stack>
                                                            <Stack spacing={1} width={32}>
                                                                <IconButton
                                                                    edge="end"
                                                                    size="small"
                                                                    onClick={() => handleDeleteStep(i)}
                                                                    sx={{ margin: "auto" }}
                                                                >
                                                                    <DeleteIcon fill={palette.error.light} />
                                                                </IconButton>
                                                                <Box
                                                                    {...provided.dragHandleProps}
                                                                >
                                                                    <DragIcon fill={palette.background.textPrimary} />
                                                                </Box>
                                                            </Stack>
                                                        </Stack>
                                                    </Box>
                                                )}
                                            </Draggable>
                                        ))
                                    }
                                    {provided.placeholder}
                                </div>
                            )}
                        </Droppable>
                        <Button
                            startIcon={<AddIcon />}
                            onClick={handleAddStep}
                            sx={{ alignSelf: "center", mt: 1 }}
                        >
                            Add Step
                        </Button>
                    </Stack>
                </BaseForm>
            </DragDropContext>
            <GridSubmitButtons
                display={display}
                errors={props.errors as any}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});
