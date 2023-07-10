import { AddIcon, DeleteIcon, DragIcon, DUMMY_ID, ListNumberIcon, Reminder, reminderValidation, Session, uuid } from "@local/shared";
import { Box, Button, Checkbox, FormControlLabel, IconButton, Stack, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { DateInput } from "components/inputs/DateInput/DateInput";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { Title } from "components/text/Title/Title";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderFormProps } from "forms/types";
import { forwardRef } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ReminderShape, shapeReminder } from "utils/shape/models/reminder";
import { ReminderItemShape } from "utils/shape/models/reminderItem";

export const reminderInitialValues = (
    session: Session | undefined,
    reminderListId: string | undefined,
    existing?: Reminder | null | undefined,
): ReminderShape => ({
    __typename: "Reminder" as const,
    id: DUMMY_ID,
    dueDate: null,
    index: 0,
    isComplete: false,
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
    description: existing?.description ?? "",
    name: existing?.name ?? "",
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

export const ReminderForm = forwardRef<BaseFormRef | undefined, ReminderFormProps>(({
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
                id: uuid(),
                index: reminderItemsField.value.length,
                name: "",
                description: "",
                dueDate: null,
            },
        ]);
    };

    const onDragEnd = (result: DropResult) => {
        if (!result.destination) return;

        const newReminderItems: ReminderItemShape[] = Array.from(reminderItemsField.value);
        const [reorderedItem] = newReminderItems.splice(result.source.index, 1);
        newReminderItems.splice(result.destination.index, 0, reorderedItem);

        // Update the index property of each item in the list
        newReminderItems.forEach((item: ReminderItemShape, index) => {
            item.index = index;
        });

        reminderItemsHelpers.setValue(newReminderItems);
    };

    return (
        <>
            <DragDropContext onDragEnd={onDragEnd}>
                <BaseForm
                    dirty={dirty}
                    display={display}
                    isLoading={isLoading}
                    maxWidth={600}
                    ref={ref}
                >
                    <FormContainer>
                        <Field
                            fullWidth
                            name="name"
                            label={t("Name")}
                            as={TextField}
                        />
                        <MarkdownInput
                            maxChars={2048}
                            maxRows={10}
                            minRows={4}
                            name="description"
                            placeholder="Description (optional)"
                            zIndex={zIndex}
                        />
                        <DateInput
                            name="dueDate"
                            label="Due date (optional)"
                            type="datetime-local"
                        />
                        {/* Steps to complete reminder */}
                        <Title
                            Icon={ListNumberIcon}
                            title="Steps"
                            variant="subheader"
                            zIndex={zIndex}
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
                                                            boxShadow: 4,
                                                            marginBottom: 2,
                                                            padding: 2,
                                                            background: palette.background.default,
                                                        }}>
                                                        <Stack
                                                            direction="row"
                                                            alignItems="flex-start"
                                                            spacing={2}
                                                        >
                                                            <Stack spacing={1} sx={{ width: "100%" }}>
                                                                <Field
                                                                    fullWidth
                                                                    name={`reminderItems[${i}].name`}
                                                                    label={t("Name")}
                                                                    as={TextField}
                                                                />
                                                                <MarkdownInput
                                                                    maxChars={2048}
                                                                    maxRows={6}
                                                                    minRows={2}
                                                                    name={`reminderItems[${i}].description`}
                                                                    placeholder={t("Description")}
                                                                    zIndex={zIndex}
                                                                />
                                                                <DateInput
                                                                    name={`reminderItems[${i}].dueDate`}
                                                                    label="Due date (optional)"
                                                                    type="datetime-local"
                                                                />
                                                                <FormControlLabel
                                                                    control={<Field
                                                                        name={`reminderItems[${i}].isComplete`}
                                                                        type="checkbox"
                                                                        as={Checkbox}
                                                                        size="small"
                                                                        color="secondary"
                                                                    />}
                                                                    label="Complete?"
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
                            variant="outlined"
                            sx={{ alignSelf: "center", mt: 1 }}
                        >
                            Add Step
                        </Button>
                    </FormContainer>
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
                zIndex={zIndex}
            />
        </>
    );
});
