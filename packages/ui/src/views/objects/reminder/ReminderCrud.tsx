import { DragDropContext, Draggable, Droppable, DropResult } from "@hello-pangea/dnd";
import { DeleteOneInput, DeleteType, DUMMY_ID, endpointsActions, endpointsReminder, LlmTask, noopSubmit, Reminder, ReminderCreateInput, ReminderItemShape, ReminderShape, ReminderUpdateInput, reminderValidation, shapeReminder, Success, uuid } from "@local/shared";
import { Box, Button, Checkbox, Divider, Grid, IconButton, Palette, Paper, Stack, styled, Typography, useTheme } from "@mui/material";
import { Field, Formik, useField } from "formik";
import { memo, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper, useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { AdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures } from "../../../components/inputs/AdvancedInput/styles.js";
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

function useReminderFormStyles(palette: Palette) {
    return useMemo(() => ({
        sectionTitle: {
            marginBottom: 1,
        },
        stepPaper: {
            marginBottom: 2,
            padding: 1.5,
        },
        dragHandle: {
            cursor: "grab",
            display: "flex",
            alignItems: "center",
            paddingRight: 1,
            color: palette.text.secondary,
        },
        deleteIconButton: {
            margin: "auto",
        },
        dividerStyle: {
            display: { xs: "flex", md: "none" },
        },
    }), [palette]);
}

// Define stable sx props outside the component to fix linter errors
const dialogSx = {
    paper: {
        width: "min(100%, 1200px)",
    },
};
const checkboxSx = { padding: 0 };
const compactClickSx = { cursor: "pointer" };

// Define stable sx for Typography outside, or use useMemo if theme-dependent
function useCompactTypographySx(isComplete: boolean) {
    return useMemo(() => ({
        flexGrow: 1,
        textDecoration: isComplete ? "line-through" : "none",
    }), [isComplete]);
}

// Constant for blur delay
const STEP_BLUR_DELAY_MS = 50;

// Props for the new ReminderStep component
interface ReminderStepProps {
    reminderItem: ReminderItemShape;
    index: number;
    isEditing: boolean;
    styles: ReturnType<typeof useReminderFormStyles>;
    onFocus: (id: string) => void;
    onBlur: (id: string, nameExists: boolean) => void;
    onCompactClick: (id: string) => void;
    onDelete: () => void;
    onUpdateField: (index: number, field: keyof ReminderItemShape, value: any) => void;
    dragHandleProps: any;
}

// Memoized ReminderStep component
const ReminderStep = memo(({
    reminderItem,
    index,
    isEditing,
    styles,
    onFocus,
    onBlur,
    onCompactClick,
    onDelete,
    onUpdateField,
    dragHandleProps,
}: ReminderStepProps) => {
    const { t } = useTranslation();
    const typographySx = useCompactTypographySx(reminderItem.isComplete);

    // --- Local state for name input to avoid re-renders on every keystroke --- 
    const [localName, setLocalName] = useState(reminderItem.name);
    // Sync local state if the prop changes from outside (e.g., initial load, drag/drop)
    useEffect(() => {
        setLocalName(reminderItem.name);
    }, [reminderItem.name]);

    const handleLocalNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setLocalName(event.target.value);
    };

    const handleLocalNameBlur = () => {
        // Update Formik state only on blur if the name has changed
        if (localName !== reminderItem.name) {
            onUpdateField(index, "name", localName);
        }
        // Trigger the general step blur handler
        handleItemBlur();
    };
    // --- End local state --- 

    // Stable handler for stopping propagation
    const handleCheckboxClick = useCallback((e: React.MouseEvent) => e.stopPropagation(), []);

    // We need stable handlers for focus/blur that know the item ID and name status
    const handleItemFocus = useCallback(() => onFocus(reminderItem.id), [onFocus, reminderItem.id]);
    // Adjusted handleItemBlur to not rely on local state directly for the check
    const handleItemBlur = useCallback(() => onBlur(reminderItem.id, !!localName), [onBlur, reminderItem.id, localName]);
    const handleCompactItemClick = useCallback(() => onCompactClick(reminderItem.id), [onCompactClick, reminderItem.id]);

    return (
        <Stack direction="row" spacing={1} alignItems="center">
            {/* Drag Handle always visible */}
            <Box {...dragHandleProps} sx={styles.dragHandle}>
                <IconCommon name="Drag" />
            </Box>

            {/* Conditionally Render Compact or Full View */}
            {isEditing ? (
                // Full Edit View
                <Stack
                    flexGrow={1}
                    spacing={1.5}
                    onFocus={handleItemFocus}
                    // onBlur is handled by individual inputs now, particularly the name input
                    tabIndex={-1}
                >
                    {/* Top row: Checkbox and Name Input */}
                    <Stack direction="row" spacing={1} alignItems="center">
                        <Field
                            name={`reminderItems[${index}].isComplete`}
                            type="checkbox"
                            as={Checkbox}
                            size="small"
                            color="primary"
                            sx={checkboxSx}
                        />
                        <TextInput
                            fullWidth
                            isRequired={true}
                            name={`reminderItems[${index}].name`}
                            label={t("Name")}
                            placeholder={t("NamePlaceholder")}
                            variant="standard"
                            size="small"
                            value={localName}
                            onChange={handleLocalNameChange}
                            onBlur={handleLocalNameBlur}
                        />
                    </Stack>

                    {/* Description Input */}
                    <Field
                        isRequired={false}
                        maxChars={2048}
                        maxRows={4}
                        minRows={1}
                        name={`reminderItems[${index}].description`}
                        label={t("Description")}
                        placeholder={t("DescriptionPlaceholder")}
                        as={RichInput}
                        onFocus={handleItemFocus}
                        onBlur={handleItemBlur}
                    />

                    {/* Due Date Input */}
                    <Field
                        isRequired={false}
                        name={`reminderItems[${index}].dueDate`}
                        label={t("DueDate")}
                        type="datetime-local"
                        as={DateInput}
                        onFocus={handleItemFocus}
                        onBlur={handleItemBlur}
                    />
                </Stack>
            ) : (
                // Compact View (name exists AND not editing)
                <Stack
                    direction="row"
                    flexGrow={1}
                    spacing={1}
                    alignItems="center"
                    onClick={handleCompactItemClick}
                    sx={compactClickSx}
                >
                    <Field
                        name={`reminderItems[${index}].isComplete`}
                        type="checkbox"
                        as={Checkbox}
                        size="small"
                        color="primary"
                        sx={checkboxSx}
                        onClick={handleCheckboxClick}
                    />
                    <Typography sx={typographySx}>
                        {reminderItem.name}
                    </Typography>
                    {/* Optional: Display Due Date compactly */}
                    {reminderItem.dueDate && (
                        <Typography variant="caption" color="text.secondary">
                            {new Date(reminderItem.dueDate).toLocaleDateString()}
                        </Typography>
                    )}
                </Stack>
            )}

            {/* Delete Button always visible */}
            <IconButton
                edge="end"
                size="small"
                onClick={onDelete}
                sx={styles.deleteIconButton}
            >
                <IconCommon name="Delete" fill="error.main" />
            </IconButton>
        </Stack>
    );
});

// Add display name for React DevTools
ReminderStep.displayName = "ReminderStep";

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
    const styles = useReminderFormStyles(palette);

    // State to track the ID of the step currently being edited
    const [editingStepId, setEditingStepId] = useState<string | null>(null);

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

    const [reminderItemsField, meta, reminderItemsHelpers] = useField<ReminderItemShape[]>("reminderItems");

    // Stable handler to update a specific field of a step
    const handleUpdateStepField = useCallback((index: number, field: keyof ReminderItemShape, value: any) => {
        const newItems = [...reminderItemsField.value];
        if (newItems[index]) {
            (newItems[index] as any)[field] = value;
            reminderItemsHelpers.setValue(newItems);
        }
    }, [reminderItemsField.value, reminderItemsHelpers]);

    // Stable handler to delete a step by index
    const handleDeleteStep = useCallback((indexToDelete: number) => {
        const newReminderItems = [...reminderItemsField.value];
        newReminderItems.splice(indexToDelete, 1);
        // Re-index subsequent items
        for (let i = indexToDelete; i < newReminderItems.length; i++) {
            newReminderItems[i].index = i;
        }
        reminderItemsHelpers.setValue(newReminderItems);
    }, [reminderItemsField.value, reminderItemsHelpers]);

    // --- Stable Handlers to pass down --- 
    const handleStepFocus = useCallback((id: string) => {
        setEditingStepId(id);
    }, []);

    // Refined blur handler: Use setTimeout
    const blurTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const handleStepBlur = useCallback((id: string, nameExists: boolean) => {
        if (blurTimeoutRef.current) {
            clearTimeout(blurTimeoutRef.current);
        }
        blurTimeoutRef.current = setTimeout(() => {
            // Check if the blurred item is still the one being edited
            // If another item gained focus, editingStepId would change via handleStepFocus
            setEditingStepId(currentEditingId => {
                if (currentEditingId === id && nameExists) {
                    return null; // Collapse if name exists and focus truly left
                }
                return currentEditingId; // Otherwise keep current state
            });
        }, STEP_BLUR_DELAY_MS); // Use constant
    }, []);

    const handleCompactClick = useCallback((id: string) => {
        setEditingStepId(id);
        // Optionally, find the input and focus it programmatically here
    }, []);

    // Clean up timeout on unmount
    useEffect(() => {
        return () => {
            if (blurTimeoutRef.current) {
                clearTimeout(blurTimeoutRef.current);
            }
        };
    }, []);
    // --- End Stable Handlers --- 

    // Create stable delete handlers for each step
    const stepDeleteHandlers = useMemo(() => {
        return reminderItemsField.value.map((_, i) => () => handleDeleteStep(i));
    }, [reminderItemsField.value, handleDeleteStep]);

    function handleAddStep() {
        const newId = uuid();
        const newIndex = reminderItemsField.value.length;
        reminderItemsHelpers.setValue([
            ...reminderItemsField.value,
            {
                id: newId,
                index: newIndex,
                isComplete: false,
                name: "",
                description: "",
                dueDate: null,
            },
        ]);
        // Automatically set the new step to editing mode
        setEditingStepId(newId);
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
            sxs={dialogSx}
        >
            <TopBar
                display={display}
                onClose={onClose}
                // Display original title so it doesn't keep changing as you type a new name
                title={firstString(getDisplay(existing).title, t(isCreate ? "CreateReminder" : "UpdateReminder"))}
                options={topBarOptions}
            />
            <DragDropContext onDragEnd={onDragEnd}>
                <BaseForm
                    display={display}
                    maxWidth={1200}
                    isLoading={isLoading}
                >
                    <Box width="100%" padding={2}>
                        <Grid container spacing={2}>
                            <Grid item xs={12} lg={6}>
                                <Typography variant="h4" sx={styles.sectionTitle}>Basic info</Typography>
                                <Box display="flex" flexDirection="column" gap={4}>
                                    <RelationshipList
                                        isEditing={true}
                                        objectType={"Reminder"}
                                    />
                                    <AdvancedInput
                                        features={nameInputFeatures}
                                        isRequired={true}
                                        name="name"
                                        title={t("Name")}
                                        placeholder={t("NamePlaceholder")}
                                    />
                                    <AdvancedInput
                                        features={detailsInputFeatures}
                                        isRequired={false}
                                        name="description"
                                        title={t("Description")}
                                        placeholder={t("DescriptionPlaceholder")}
                                    />
                                    <DateInput
                                        isRequired={false}
                                        name="dueDate"
                                        label={t("DueDate")}
                                        type="datetime-local"
                                    />
                                </Box>
                            </Grid>
                            <Grid item xs={12} lg={6}>
                                <Divider sx={styles.dividerStyle} />
                                <Typography variant="h4" sx={styles.sectionTitle}>Steps to complete</Typography>
                                <Box>
                                    <Droppable droppableId="reminderItems">
                                        {(provided) => (
                                            <div ref={provided.innerRef} {...provided.droppableProps}>
                                                {
                                                    (reminderItemsField.value ?? []).map((reminderItem, i) => (
                                                        <Draggable key={reminderItem.id} draggableId={String(reminderItem.id)} index={i}>
                                                            {(providedDraggable) => (
                                                                <Paper
                                                                    ref={providedDraggable.innerRef}
                                                                    {...providedDraggable.draggableProps}
                                                                    sx={styles.stepPaper}
                                                                    elevation={2}
                                                                >
                                                                    <ReminderStep
                                                                        reminderItem={reminderItem}
                                                                        index={i}
                                                                        isEditing={!reminderItem.name || editingStepId === reminderItem.id}
                                                                        styles={styles}
                                                                        onFocus={handleStepFocus}
                                                                        onBlur={handleStepBlur}
                                                                        onCompactClick={handleCompactClick}
                                                                        onDelete={stepDeleteHandlers[i]}
                                                                        onUpdateField={handleUpdateStepField}
                                                                        dragHandleProps={providedDraggable.dragHandleProps}
                                                                    />
                                                                </Paper>
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
                                </Box>
                            </Grid>
                        </Grid>
                    </Box>
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
                disabled={false}
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
