import { DUMMY_ID, Reminder, reminderValidation, Session, uuid } from "@local/shared";
import { Box, Button, Checkbox, FormControlLabel, IconButton, Stack, TextField, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { DateInput } from "components/inputs/DateInput/DateInput";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { Title } from "components/text/Title/Title";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderFormProps } from "forms/types";
import { AddIcon, DeleteIcon, DragIcon, ListNumberIcon } from "icons";
import { forwardRef } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ReminderShape, shapeReminder } from "utils/shape/models/reminder";
import { ReminderItemShape } from "utils/shape/models/reminderItem";

export type NewReminderShape = Partial<Omit<Reminder, "reminderList">> & { reminderList: Partial<Reminder["reminderList"]> & { id: string } };

export const reminderInitialValues = (
    session: Session | undefined,
    existing?: Partial<NewReminderShape> | null | undefined,
): ReminderShape => ({
    __typename: "Reminder" as const,
    id: DUMMY_ID,
    dueDate: null,
    index: 0,
    isComplete: false,
    reminderItems: [],
    ...existing,
    reminderList: {
        __typename: "ReminderList" as const,
        id: existing?.reminderList?.id ?? DUMMY_ID,
        // If there's no reminderListId, add additional fields to create a new reminderList
        ...(existing?.reminderList?.id === undefined && {
            focusMode: getCurrentUser(session)?.activeFocusMode?.mode,
        }),
        ...existing?.reminderList,
    },
    description: existing?.description ?? "",
    name: existing?.name ?? "",
});

export const transformReminderValues = (values: ReminderShape, existing: ReminderShape, isCreate: boolean) =>
    isCreate ? shapeReminder.create(values) : shapeReminder.update(existing, values);

export const validateReminderValues = async (values: ReminderShape, existing: ReminderShape, isCreate: boolean) => {
    const transformedValues = transformReminderValues(values, existing, isCreate);
    const validationSchema = reminderValidation[isCreate ? "create" : "update"]({});
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
    ...props
}, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

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
                        <RelationshipList
                            isEditing={true}
                            objectType={"Reminder"}
                        />
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
                            placeholder={t("DescriptionOptional")}
                        />
                        <DateInput
                            name="dueDate"
                            label={t("DueDateOptional")}
                            type="datetime-local"
                        />
                        {/* Steps to complete reminder */}
                        <Box display="flex" flexDirection="column">
                            <Title
                                Icon={ListNumberIcon}
                                title={t("Step", { count: 2 })}
                                variant="subheader"
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
                                                                    />
                                                                    <DateInput
                                                                        name={`reminderItems[${i}].dueDate`}
                                                                        label={t("DueDateOptional")}
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
                                                                        label={t("CompleteQuestion")}
                                                                    />
                                                                </Stack>
                                                                <Stack spacing={1} width={32} sx={{ justifyContent: "center", alignItems: "center" }}>
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
                                {t("StepAdd")}
                            </Button>
                        </Box>
                    </FormContainer>
                </BaseForm>
            </DragDropContext>
            <BottomActionsButtons
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
