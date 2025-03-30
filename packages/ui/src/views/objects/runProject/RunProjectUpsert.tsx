import { DUMMY_ID, endpointsRunProject, noopSubmit, RunProject, RunProjectCreateInput, RunProjectShape, RunProjectUpdateInput, runProjectValidation, RunStatus, Schedule, Session, shapeRunProject } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { Formik, useField } from "formik";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { ListContainer } from "../../../components/containers/ListContainer.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ScheduleUpsert } from "../../../views/objects/schedule/ScheduleUpsert.js";
import { RunProjectFormProps, RunProjectUpsertProps } from "./types.js";

export function runProjectInitialValues(
    session: Session | undefined,
    existing?: Partial<RunProject> | null | undefined,
): RunProjectShape {
    return {
        __typename: "RunProject" as const,
        id: DUMMY_ID,
        completedComplexity: 0,
        contextSwitches: 0,
        isPrivate: true,
        name: existing?.name ?? getDisplay(existing?.projectVersion, getUserLanguages(session)).title ?? "Run",
        schedule: null,
        status: RunStatus.Scheduled,
        steps: [],
        timeElapsed: 0,
        ...existing,
    };
}

export function transformRunProjectValues(values: RunProjectShape, existing: RunProjectShape, isCreate: boolean) {
    return isCreate ? shapeRunProject.create(values) : shapeRunProject.update(existing, values);
}

function RunProjectForm({
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
}: RunProjectFormProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle scheduling
    const [scheduleField, , scheduleHelpers] = useField<Schedule | null>("schedule");
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    function handleAddSchedule() {
        setIsScheduleDialogOpen(true);
    }
    function handleUpdateSchedule() {
        setEditingSchedule(scheduleField.value);
        setIsScheduleDialogOpen(true);
    }
    function handleCloseScheduleDialog() {
        setIsScheduleDialogOpen(false);
    }
    function handleScheduleCompleted(created: Schedule) {
        scheduleHelpers.setValue(created);
        setIsScheduleDialogOpen(false);
    }
    function handleScheduleDeleted() {
        scheduleHelpers.setValue(null);
        setIsScheduleDialogOpen(false);
    }

    const { handleCancel, handleCompleted } = useUpsertActions<RunProject>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "RunProject",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<RunProject, RunProjectCreateInput, RunProjectUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsRunProject.createOne,
        endpointUpdate: endpointsRunProject.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "RunProject" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<RunProjectCreateInput | RunProjectUpdateInput, RunProject>({
        disabled,
        existing,
        fetch,
        inputs: transformRunProjectValues(values, existing, isCreate),
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
            {/* Dialog to create/update schedule */}
            <ScheduleUpsert
                canSetScheduleFor={false}
                defaultScheduleFor="RunProject"
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
                            startIcon={<IconCommon name="Add" />}
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
                                        <IconCommon name="Edit" fill="secondary.main" />
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
                                        <IconCommon name="Delete" fill="secondary.main" />
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

export function RunProjectUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: RunProjectUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<RunProject, RunProjectShape>({
        ...endpointsRunProject.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "RunProject",
        overrideObject,
        transform: (existing) => runProjectInitialValues(session, existing),
    });

    async function validateValues(values: RunProjectShape) {
        return await validateFormValues(values, existing, isCreate, transformRunProjectValues, runProjectValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <RunProjectForm
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
