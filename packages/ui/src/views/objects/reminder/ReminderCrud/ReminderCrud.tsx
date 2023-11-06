import { DeleteOneInput, DeleteType, DUMMY_ID, endpointGetReminder, endpointPostDeleteOne, endpointPostReminder, endpointPutReminder, noopSubmit, Reminder, ReminderCreateInput, ReminderUpdateInput, reminderValidation, Session, Success, uuid } from "@local/shared";
import { Box, Button, Checkbox, FormControlLabel, IconButton, Stack, TextField, useTheme } from "@mui/material";
import { fetchLazyWrapper, useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { DateInput } from "components/inputs/DateInput/DateInput";
import { RichInput } from "components/inputs/RichInput/RichInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, DeleteIcon, DragIcon, ListNumberIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getFocusModeInfo } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { ReminderShape, shapeReminder } from "utils/shape/models/reminder";
import { ReminderItemShape } from "utils/shape/models/reminderItem";
import { validateFormValues } from "utils/validateFormValues";
import { ReminderCrudProps, ReminderFormProps } from "../types";

export type NewReminderShape = Partial<Omit<Reminder, "reminderList">> & { reminderList: Partial<Reminder["reminderList"]> & { id: string } };

const getFallbackReminderList = (session: Session | undefined, existing: Partial<NewReminderShape> | null | undefined) => {
    const { active: activeFocusMode, all: allFocusModes } = getFocusModeInfo(session);
    const activeMode = activeFocusMode?.mode;

    // If reminderList exists, return it
    if (existing?.reminderList) {
        // Try to add the relevant focus mode to the reminder list so we can display it
        const focusModeId = existing.reminderList.focusMode?.id;
        return {
            ...existing.reminderList,
            __typename: "ReminderList" as const,
            focusMode: focusModeId ? allFocusModes.find(f => f.id === focusModeId) : undefined,
        };
    }
    // If active mode exists, return it
    else if (activeMode?.id) {
        return {
            __typename: "ReminderList" as const,
            ...activeMode.reminderList,
            id: activeMode.reminderList?.id ?? DUMMY_ID,
            focusMode: activeMode,
        };
    }
    // Otherwise, return a new reminder list with any existing focus mode
    const focusWithReminderList = allFocusModes.find(f => f.reminderList?.id);
    return {
        __typename: "ReminderList" as const,
        id: focusWithReminderList?.reminderList?.id ?? DUMMY_ID,
        focusMode: focusWithReminderList,
    };
};

const reminderInitialValues = (
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
    reminderList: existing?.reminderList || getFallbackReminderList(session, existing),
    description: existing?.description ?? "",
    name: existing?.name ?? "",
});

const transformReminderValues = (values: ReminderShape, existing: ReminderShape, isCreate: boolean) =>
    isCreate ? shapeReminder.create(values) : shapeReminder.update(existing, values);

const ReminderForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    index,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    reminderListId,
    values,
    ...props
}: ReminderFormProps) => {
    const display = toDisplay(isOpen);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { handleCancel, handleCreated, handleCompleted, handleDeleted } = useUpsertActions<Reminder>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Reminder",
        ...props,
    });
    const {
        fetch,
        fetchCreate,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Reminder, ReminderCreateInput, ReminderUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostReminder,
        endpointUpdate: endpointPutReminder,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Reminder" });

    const onSubmit = useSubmitHelper<ReminderCreateInput | ReminderUpdateInput, Reminder>({
        disabled,
        existing,
        fetch,
        inputs: transformReminderValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    // Handle delete
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback(() => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteMutation,
            inputs: { id: values.id, objectType: DeleteType.Reminder },
            successCondition: (data) => data.success,
            successMessage: () => ({
                messageKey: "ObjectDeleted",
                messageVariables: { objectName: values.name },
                buttonKey: "Undo",
                buttonClicked: () => {
                    fetchLazyWrapper<ReminderCreateInput, Reminder>({
                        fetch: fetchCreate,
                        inputs: transformReminderValues({
                            ...values,
                            // Make sure not to set any extra fields, 
                            // so this is treated as a "Connect" instead of a "Create"
                            reminderList: {
                                __typename: "ReminderList" as const,
                                id: values.reminderList.id,
                            },
                        }, values, true) as ReminderCreateInput,
                        successCondition: (data) => !!data.id,
                        onSuccess: (data) => { handleCreated(data); },
                    });
                },
            }),
            onSuccess: () => {
                handleDeleted(values as Reminder);
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
        });
    }, [deleteMutation, fetchCreate, handleCreated, handleDeleted, values]);

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || isDeleteLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isDeleteLoading, isUpdateLoading, props.isSubmitting]);

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
                isComplete: false,
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
        <MaybeLargeDialog
            display={display}
            id="reminder-crud-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(getDisplay(values).title, t(isCreate ? "CreateReminder" : "UpdateReminder"))}
                // Show delete button only when updating
                options={!isCreate ? [{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: handleDelete,
                }] : []}
            />
            <DragDropContext onDragEnd={onDragEnd}>
                <BaseForm
                    display={display}
                    isLoading={isLoading}
                    maxWidth={700}
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
                        <RichInput
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
                                                                background: palette.mode === "light" ? palette.background.default : palette.background.paper,
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
                                                                    <RichInput
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
                errors={props.errors}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const ReminderCrud = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ReminderCrudProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Reminder, ReminderShape>({
        ...endpointGetReminder,
        isCreate,
        objectType: "Reminder",
        overrideObject: overrideObject as Reminder,
        transform: (existing) => reminderInitialValues(session, existing),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformReminderValues, reminderValidation)}
        >
            {(formik) => <ReminderForm
                disabled={false} // Can always update reminders
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
