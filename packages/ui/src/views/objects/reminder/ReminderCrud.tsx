import { DragDropContext, Draggable, Droppable, DropResult } from "@hello-pangea/dnd";
import { DeleteOneInput, DeleteType, DUMMY_ID, endpointsActions, endpointsReminder, LlmTask, noopSubmit, Reminder, ReminderCreateInput, ReminderItemShape, ReminderShape, ReminderUpdateInput, reminderValidation, shapeReminder, Success, uuid } from "@local/shared";
import { Box, Button, Checkbox, Divider, FormControlLabel, IconButton, Stack, styled, useTheme } from "@mui/material";
import { Field, Formik, useField } from "formik";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper, useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { DateInput } from "../../../components/inputs/DateInput/DateInput.js";
import { RichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useAutoFill, UseAutoFillProps } from "../../../hooks/tasks.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FocusModeInfo, useFocusModes } from "../../../stores/focusModeStore.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ReminderCrudProps, ReminderFormProps } from "./types.js";

function getFallbackReminderList(focusModeInfo: FocusModeInfo, existing: Partial<ReminderShape> | null | undefined) {
    const { active: activeFocusMode, all: allFocusModes } = focusModeInfo;
    const activeMode = activeFocusMode?.focusMode;

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
            id: activeMode.reminderListId ?? DUMMY_ID,
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
}

export function reminderInitialValues(
    focusModeInfo: FocusModeInfo,
    existing?: Partial<ReminderShape> | null | undefined,
): ReminderShape {
    return {
        __typename: "Reminder" as const,
        id: DUMMY_ID,
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderItems: [],
        ...existing,
        reminderList: existing?.reminderList || getFallbackReminderList(focusModeInfo, existing),
        description: existing?.description ?? "",
        name: existing?.name ?? "",
    };
}

function transformReminderValues(values: ReminderShape, existing: ReminderShape, isCreate: boolean) {
    return isCreate ? shapeReminder.create(values) : shapeReminder.update(existing, values);
}

const AddStepButton = styled(Button)(({ theme }) => ({
    alignSelf: "center",
    marginTop: theme.spacing(1),
}));

const relationshipListStyle = { marginBottom: 4 } as const;
const formSectionStyle = { overflowX: "hidden" } as const;

function ReminderForm({
    disabled,
    dirty,
    display,
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
}: ReminderFormProps) {
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
        endpointCreate: endpointsReminder.createOne,
        endpointUpdate: endpointsReminder.updateOne,
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

    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
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

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        const input = {
            description: values.description,
            dueDate: values.dueDate,
            isComplete: values.isComplete,
            name: values.name,
            steps: values.reminderItems?.map(step => ({
                description: step.description,
                dueDate: step.dueDate,
                isComplete: step.isComplete,
                name: step.name,
            })) ?? [],
        } as Record<string, unknown> & { steps?: Record<string, unknown>[] };
        if (input.steps && input.steps.length === 0) delete input.steps;
        return input;
    }, [values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        if (!data || typeof data !== "object") return { originalValues, updatedValues: originalValues };
        const dataObject = data as Record<string, unknown>;
        const updatedValues = {
            ...values,
            description: typeof dataObject.description === "string" ? dataObject.description : values.description,
            dueDate: typeof dataObject.dueDate === "string" ? dataObject.dueDate : values.dueDate,
            isComplete: typeof dataObject.isComplete === "boolean" ? dataObject.isComplete : values.isComplete,
            name: typeof dataObject.name === "string" ? dataObject.name : values.name,
            reminderItems: Array.isArray(dataObject.steps) ? dataObject.steps.map((step, i) => ({
                //...values.reminderItems?.[i],
                id: DUMMY_ID,
                index: i,
                isComplete: step.isComplete ?? false,
                name: step.name ?? "",
                description: step.description ?? "",
                dueDate: step.dueDate ?? null,
            })) : values.reminderItems,
        } as ReminderShape;
        return { originalValues, updatedValues };
    }, [values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.ReminderAdd : LlmTask.ReminderUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || isDeleteLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, isDeleteLoading, props.isSubmitting]);

    const [reminderItemsField, , reminderItemsHelpers] = useField("reminderItems");

    function handleDeleteStep(index: number) {
        const newReminderItems = [...reminderItemsField.value];
        newReminderItems.splice(index, 1);
        reminderItemsHelpers.setValue(newReminderItems);
    }

    function handleAddStep() {
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
    }

    function onDragEnd(result: DropResult) {
        if (!result.destination) return;

        const newReminderItems: ReminderItemShape[] = Array.from(reminderItemsField.value);
        const [reorderedItem] = newReminderItems.splice(result.source.index, 1);
        newReminderItems.splice(result.destination.index, 0, reorderedItem);

        // Update the index property of each item in the list
        newReminderItems.forEach((item: ReminderItemShape, index) => {
            item.index = index;
        });

        reminderItemsHelpers.setValue(newReminderItems);
    }

    const topBarOptions = useMemo(function topBarOptionsMemo() {
        return !isCreate ? [{
            iconInfo: { name: "Delete", type: "Common" } as const,
            label: t("Delete"),
            onClick: handleDelete,
        }] : [];
    }, [handleDelete, isCreate, t]);

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
                options={topBarOptions}
            />
            <DragDropContext onDragEnd={onDragEnd}>
                <BaseForm
                    display={display}
                    isLoading={isLoading}
                >
                    <FormContainer>
                        <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                            <RelationshipList
                                isEditing={true}
                                objectType={"Reminder"}
                                sx={relationshipListStyle}
                            />
                            <FormSection sx={formSectionStyle}>
                                <Field
                                    fullWidth
                                    isRequired={true}
                                    name="name"
                                    label={t("Name")}
                                    placeholder={t("NamePlaceholder")}
                                    as={TextInput}
                                />
                                <RichInput
                                    isRequired={false}
                                    maxChars={2048}
                                    maxRows={10}
                                    minRows={4}
                                    name="description"
                                    label={t("Description")}
                                    placeholder={t("DescriptionPlaceholder")}
                                />
                                <DateInput
                                    isRequired={false}
                                    name="dueDate"
                                    label={t("DueDate")}
                                    type="datetime-local"
                                />
                            </FormSection>
                        </ContentCollapse>
                        <Divider />
                        <ContentCollapse title="Steps to complete" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
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
                                                                background: palette.background.paper,
                                                            }}>
                                                            <Stack
                                                                direction="row"
                                                                alignItems="flex-start"
                                                                spacing={2}
                                                            >
                                                                <Stack spacing={1} sx={{ width: "100%" }}>
                                                                    <Field
                                                                        fullWidth
                                                                        isRequired={true}
                                                                        name={`reminderItems[${i}].name`}
                                                                        label={t("Name")}
                                                                        placeholder={t("NamePlaceholder")}
                                                                        as={TextInput}
                                                                    />
                                                                    <RichInput
                                                                        isRequired={false}
                                                                        maxChars={2048}
                                                                        maxRows={6}
                                                                        minRows={2}
                                                                        name={`reminderItems[${i}].description`}
                                                                        label={t("Description")}
                                                                        placeholder={t("DescriptionPlaceholder")}
                                                                    />
                                                                    <DateInput
                                                                        isRequired={false}
                                                                        name={`reminderItems[${i}].dueDate`}
                                                                        label={t("DueDate")}
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
                                                                        <IconCommon name="Delete" fill="error.light" />
                                                                    </IconButton>
                                                                    <Box
                                                                        {...provided.dragHandleProps}
                                                                    >
                                                                        <IconCommon name="Drag" fill="background.textPrimary" />
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
                            <AddStepButton
                                startIcon={<IconCommon name="Add" />}
                                onClick={handleAddStep}
                                variant="outlined"
                            >
                                {t("StepAdd")}
                            </AddStepButton>
                        </ContentCollapse>
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
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
        </MaybeLargeDialog>
    );
}

export function ReminderCrud({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ReminderCrudProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useManagedObject<Reminder, ReminderShape>({
        ...endpointsReminder.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "Reminder",
        overrideObject: overrideObject as Reminder,
        // Initially, don't transform. We'll transform after we have focusModeInfo.
        transform: (existing) => existing as ReminderShape,
    });

    const focusModeInfo = useFocusModes(session);
    useEffect(function setFocusModeInfoEffect() {
        setExisting(existing => reminderInitialValues(focusModeInfo, existing));
    }, [focusModeInfo, setExisting, isCreate]);

    async function validateValues(values: ReminderShape) {
        return await validateFormValues(values, existing, isCreate, transformReminderValues, reminderValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ReminderForm
                disabled={false} // Can always update reminders
                display={display}
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
}
