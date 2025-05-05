import { DragDropContext, Draggable, Droppable, DropResult } from "@hello-pangea/dnd";
import { CanConnect, DeleteOneInput, DeleteType, DUMMY_ID, endpointsActions, endpointsReminder, LlmTask, noopSubmit, Reminder, ReminderCreateInput, ReminderItemShape, ReminderListShape, ReminderShape, ReminderUpdateInput, reminderValidation, shapeReminder, Success, uuid } from "@local/shared";
import { Box, Button, Checkbox, Divider, Grid, IconButton, InputBase, Palette, Paper, Stack, styled, Typography, useTheme } from "@mui/material";
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
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useAutoFill, UseAutoFillProps } from "../../../hooks/tasks.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ReminderCrudProps, ReminderFormProps } from "./types.js";

export function reminderInitialValues(existing?: Partial<ReminderShape> & { reminderList: CanConnect<ReminderListShape> } | null | undefined): ReminderShape {
    return {
        __typename: "Reminder" as const,
        id: DUMMY_ID,
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderItems: [],
        ...existing,
        reminderList: existing?.reminderList,
        description: existing?.description ?? "",
        name: existing?.name ?? "",
    };
}

function transformReminderValues(values: ReminderShape, existing: ReminderShape, isCreate: boolean) {
    return isCreate ? shapeReminder.create(values) : shapeReminder.update(existing, values);
}

const AddButton = styled(Button)(({ theme }) => ({
    alignSelf: "center",
    backgroundColor: theme.palette.background.paper,
    borderRadius: 0,
    color: theme.palette.background.textSecondary,
}));

function useReminderFormStyles(palette: Palette) {
    return useMemo(() => ({
        sectionTitle: {
            marginBottom: 1,
        },
        stepPaper: {
            background: palette.background.paper,
            borderBottom: `1px solid ${palette.divider}`,
            borderRadius: 0,
            padding: 1.5,
            position: "relative",
        },
        dragHandle: {
            cursor: "grab",
            display: "flex",
            alignItems: "center",
            paddingRight: 1,
            color: palette.text.secondary,
        },
        deleteIconButton: {
            position: "absolute",
            top: 8,
            right: 8,
            zIndex: 1,
        },
        dividerStyle: {
            display: { xs: "flex", lg: "none" },
            marginBottom: { xs: 2, lg: 0 },
            marginTop: { xs: 2, lg: 0 },
        },
    }), [palette]);
}

const dialogSx = {
    paper: {
        width: "min(100%, 1200px)",
    },
};

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
    onFocus: (index: number) => void;
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
    onDelete,
    onUpdateField,
    dragHandleProps,
}: ReminderStepProps) => {
    const { t } = useTranslation();
    const typographySx = useCompactTypographySx(reminderItem.isComplete);

    // Create a ref for the DateInput
    const dateInputRef = useRef<HTMLInputElement>(null);

    const [localName, setLocalName] = useState(reminderItem.name);
    // Sync local state if the prop changes from outside (e.g., initial load, drag/drop)
    useEffect(() => {
        setLocalName(reminderItem.name);
    }, [reminderItem.name]);

    const handleLocalNameChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setLocalName(event.target.value);
    }, []);

    // We need stable handlers for focus/blur that know the item ID and name status
    const handleItemFocus = useCallback(() => onFocus(index), [onFocus, index]);

    const handleLocalNameBlur = useCallback(() => {
        // Update Formik state only on blur if the name has changed
        if (localName !== reminderItem.name) {
            onUpdateField(index, "name", localName);
        }
    }, [index, localName, onUpdateField, reminderItem.name]);

    // Handler to clear the date
    const handleClearDate = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
        onUpdateField(index, "dueDate", null);
    }, [index, onUpdateField]);

    // Handler for date change
    const handleDateChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        if (e.target.value) {
            onUpdateField(index, "dueDate", new Date(e.target.value).toISOString());
        }
    }, [index, onUpdateField]);

    // Click handler just activates the input
    const handleDateClick = useCallback(() => {
        if (dateInputRef.current) {
            dateInputRef.current.showPicker();
        }
    }, []);

    return (
        <Stack direction="row" spacing={1} alignItems="center" sx={{ position: 'relative' }}>
            {/* Drag Handle always visible */}
            <Box {...dragHandleProps} sx={styles.dragHandle}>
                <IconCommon name="Drag" />
            </Box>

            <Stack
                flexGrow={1}
                spacing={1.5}
                onFocus={handleItemFocus}
                // onBlur is handled by individual inputs now, particularly the name input
                tabIndex={-1}
            >
                {/* Top row: Checkbox, Name Input, Date, and Delete Button */}
                <Stack
                    direction="row"
                    spacing={1}
                    alignItems="center"
                >
                    <Field
                        name={`reminderItems[${index}].isComplete`}
                        type="checkbox"
                        as={Checkbox}
                        size="small"
                        color="secondary"
                    />
                    <InputBase
                        fullWidth
                        required={true}
                        name={`reminderItems[${index}].name`}
                        placeholder={t("NamePlaceholder")}
                        value={localName}
                        onChange={handleLocalNameChange}
                        onBlur={handleLocalNameBlur}
                        sx={{
                            fontSize: 'inherit',
                            '& .MuiInputBase-input': {
                                padding: 0,
                                height: 'auto',
                            },
                            '&.Mui-focused': {
                            },
                            '&:before': { borderBottom: 'none' },
                            '&:hover:not(.Mui-disabled):before': { borderBottom: 'none' },
                            '&:after': { borderBottom: 'none' },
                        }}
                    />

                    {/* Date and Delete controls */}
                    <Stack
                        direction="row"
                        spacing={0.5}
                        alignItems="center"
                    >
                        {/* Conditionally render Due Date display/trigger only in editing mode */}
                        {(isEditing || reminderItem.dueDate) && (
                            <>
                                {/* Wrapper to position both the text and input */}
                                <Box sx={{ position: 'relative', width: "max-content" }}>
                                    {/* The visible clickable text */}
                                    <Typography
                                        variant="caption"
                                        onClick={handleDateClick}
                                        sx={{
                                            cursor: "pointer",
                                            color: "text.secondary",
                                            lineHeight: 'normal',
                                            borderBottom: '1px dashed',
                                            borderColor: 'text.secondary',
                                            padding: '2px 4px',
                                            position: 'relative',
                                            zIndex: 1, // Keep text above input
                                        }}
                                    >
                                        {reminderItem.dueDate
                                            ? new Date(reminderItem.dueDate).toLocaleDateString()
                                            : t("DateAdd")
                                        }
                                    </Typography>

                                    {/* Native date input positioned directly on top of the text */}
                                    <input
                                        ref={dateInputRef}
                                        type="datetime-local"
                                        value={reminderItem.dueDate
                                            ? new Date(reminderItem.dueDate)
                                                .toISOString()
                                                .substring(0, new Date(reminderItem.dueDate).toISOString().lastIndexOf(':'))
                                            : ''}
                                        onChange={handleDateChange}
                                        style={{
                                            position: 'absolute',
                                            top: 0,
                                            left: 0,
                                            width: '100%',
                                            height: '100%',
                                            opacity: 0, // Invisible but clickable
                                            cursor: 'pointer',
                                        }}
                                    />
                                </Box>

                                {/* Clear Date Button */}
                                {reminderItem.dueDate && (
                                    <IconButton
                                        edge="end"
                                        size="small"
                                        onClick={handleClearDate}
                                        sx={{ padding: '2px', marginLeft: '2px' }}
                                    >
                                        <IconCommon name="Close" size={14} fill="text.secondary" />
                                    </IconButton>
                                )}
                            </>
                        )}
                        <IconButton
                            edge="end"
                            size="small"
                            onClick={onDelete}
                            sx={{ padding: '2px' }}
                        >
                            <IconCommon name="Delete" fill="error.main" />
                        </IconButton>
                    </Stack>
                </Stack>

                {/* Description Input: Show if editing OR description exists */}
                {(isEditing || reminderItem.description) && (
                    <Field
                        name={`reminderItems[${index}].description`}
                        isRequired={false}
                        placeholder={t("DescriptionPlaceholder")}
                        maxRows={4}
                        minRows={1}
                        multiline
                        fullWidth
                        as={InputBase}
                        onFocus={handleItemFocus}
                        sx={{
                            fontSize: 'inherit',
                            padding: '4px 0 5px',
                            '& .MuiInputBase-input': {
                                padding: 0,
                                height: 'auto',
                                lineHeight: 1.43,
                            },
                            '&.Mui-focused': {},
                            '&:before': { borderBottom: 'none' },
                            '&:hover:not(.Mui-disabled):before': { borderBottom: 'none' },
                            '&:after': { borderBottom: 'none' },
                        }}
                    />
                )}
            </Stack>
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
    const [focusedStepIndex, setFocusedStepIndex] = useState<number | null>(null);

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
    const handleStepFocusByIndex = useCallback((index: number) => {
        setFocusedStepIndex(index);
    }, []);

    function handleAddStep() {
        const newId = uuid();
        // Use default array for safety when accessing length or spreading
        const currentItems = reminderItemsField.value ?? [];
        const newIndex = currentItems.length;
        reminderItemsHelpers.setValue([
            ...currentItems,
            {
                __typename: "ReminderItem" as const, // Add __typename
                id: newId,
                index: newIndex,
                isComplete: false,
                name: "",
                description: "",
                dueDate: null,
                reminder: { id: values.id, __typename: "Reminder" as const },
            },
        ]);
        // Automatically set the new step to editing mode
        setFocusedStepIndex(newIndex);
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
                                <Box display="flex" flexDirection="column" gap={4} maxWidth="600px" margin="auto">
                                    <Typography variant="h5" sx={styles.sectionTitle}>Basic info</Typography>
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
                                <Box display="flex" flexDirection="column" gap={4} justifyContent="flex-start" maxWidth="600px" margin="auto">
                                    <Typography variant="h5" sx={styles.sectionTitle}>Steps to complete</Typography>
                                    <Box borderRadius="24px" overflow="overlay">
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
                                                                            isEditing={focusedStepIndex === i}
                                                                            styles={styles}
                                                                            onFocus={handleStepFocusByIndex}
                                                                            onDelete={() => handleDeleteStep(i)}
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
                                        <AddButton
                                            fullWidth
                                            startIcon={<IconCommon name="Add" />}
                                            onClick={handleAddStep}
                                            variant="contained"
                                        >
                                            {t("StepAdd")}
                                        </AddButton>
                                    </Box>
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
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "Reminder",
        overrideObject: overrideObject as Reminder,
        transform: (existing) => reminderInitialValues(existing),
    });

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
