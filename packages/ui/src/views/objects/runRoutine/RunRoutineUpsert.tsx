import { DUMMY_ID, endpointsRunRoutine, noopSubmit, RunRoutine, RunRoutineCreateInput, RunRoutineShape, RunRoutineUpdateInput, runRoutineValidation, RunStatus, Schedule, Session, shapeRunRoutine } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { Formik, useField } from "formik";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { ListContainer } from "../../../components/containers/ListContainer/ListContainer.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { AddIcon, DeleteIcon, EditIcon } from "../../../icons/common.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ScheduleUpsert } from "../schedule/ScheduleUpsert.js";
import { RunRoutineFormProps, RunRoutineUpsertProps } from "./types.js";

export function runRoutineInitialValues(
    session: Session | undefined,
    existing?: Partial<RunRoutine> | null | undefined,
): RunRoutineShape {
    return {
        __typename: "RunRoutine" as const,
        id: DUMMY_ID,
        completedComplexity: 0,
        contextSwitches: 0,
        isPrivate: true,
        name: existing?.name ?? getDisplay(existing?.routineVersion, getUserLanguages(session)).title ?? "Run",
        schedule: null,
        status: RunStatus.Scheduled,
        steps: [],
        timeElapsed: 0,
        ...existing,
    };
}

export function transformRunRoutineValues(values: RunRoutineShape, existing: RunRoutineShape, isCreate: boolean) {
    return isCreate ? shapeRunRoutine.create(values) : shapeRunRoutine.update(existing, values);
}

function RunRoutineForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: RunRoutineFormProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle scheduling
    const [scheduleField, , scheduleHelpers] = useField<Schedule | null>("schedule");
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    const handleAddSchedule = () => { setIsScheduleDialogOpen(true); };
    const handleUpdateSchedule = () => {
        setEditingSchedule(scheduleField.value);
        setIsScheduleDialogOpen(true);
    };
    const handleCloseScheduleDialog = () => { setIsScheduleDialogOpen(false); };
    const handleScheduleCompleted = (created: Schedule) => {
        scheduleHelpers.setValue(created);
        setIsScheduleDialogOpen(false);
    };
    const handleScheduleDeleted = () => {
        scheduleHelpers.setValue(null);
        setIsScheduleDialogOpen(false);
    };

    const { handleCancel, handleCompleted } = useUpsertActions<RunRoutine>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "RunRoutine",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsRunRoutine.createOne,
        endpointUpdate: endpointsRunRoutine.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "RunRoutine" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<RunRoutineCreateInput | RunRoutineUpdateInput, RunRoutine>({
        disabled,
        existing,
        fetch,
        inputs: transformRunRoutineValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="run-routine-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateRun" : "UpdateRun")}
            />
            <ScheduleUpsert
                canSetScheduleFor={false}
                defaultScheduleFor="RunRoutine"
                display="dialog"
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onClose={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                onDeleted={handleScheduleDeleted}
                overrideObject={editingSchedule ?? { __typename: "Schedule" }}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={600}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {/* TODO */}
                    {/* Handle adding, updating, and removing schedule */}
                    {!scheduleField.value && (
                        <Button
                            onClick={handleAddSchedule}
                            startIcon={<AddIcon />}
                            variant="outlined"
                            sx={{
                                display: "flex",
                                margin: "auto",
                            }}
                        >{t("ScheduleCreate")}</Button>
                    )}
                    {scheduleField.value && <ListContainer
                        isEmpty={false}
                    >
                        {scheduleField.value && (
                            <ListItem>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    pl={2}
                                    sx={{
                                        width: "-webkit-fill-available",
                                        display: "grid",
                                        pointerEvents: "none",
                                    }}
                                >
                                    {/* TODO */}
                                </Stack>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    sx={{
                                        pointerEvents: "none",
                                        justifyContent: "center",
                                        alignItems: "start",
                                    }}
                                >
                                    {/* Edit */}
                                    <Box
                                        component="a"
                                        onClick={handleUpdateSchedule}
                                        sx={{
                                            display: "flex",
                                            justifyContent: "center",
                                            alignItems: "center",
                                            cursor: "pointer",
                                            pointerEvents: "all",
                                            paddingBottom: "4px",
                                        }}>
                                        <EditIcon fill={palette.secondary.main} />
                                    </Box>
                                    {/* Delete */}
                                    <Box
                                        component="a"
                                        onClick={handleScheduleDeleted}
                                        sx={{
                                            display: "flex",
                                            justifyContent: "center",
                                            alignItems: "center",
                                            cursor: "pointer",
                                            pointerEvents: "all",
                                            paddingBottom: "4px",
                                        }}>
                                        <DeleteIcon fill={palette.secondary.main} />
                                    </Box>
                                </Stack>
                            </ListItem>
                        )}
                    </ListContainer>}
                    {/* TODO */}
                </Stack>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
}

export function RunRoutineUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: RunRoutineUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<RunRoutine, RunRoutineShape>({
        ...endpointsRunRoutine.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "RunRoutine",
        overrideObject,
        transform: (existing) => runRoutineInitialValues(session, existing),
    });

    async function validateValues(values: RunRoutineShape) {
        return await validateFormValues(values, existing, isCreate, transformRunRoutineValues, runRoutineValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <RunRoutineForm
                disabled={!(isCreate || permissions.canUpdate)}
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
