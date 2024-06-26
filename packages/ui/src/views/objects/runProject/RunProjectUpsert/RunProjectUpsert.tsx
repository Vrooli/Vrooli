import { DUMMY_ID, endpointGetRunProject, endpointPostRunProject, endpointPutRunProject, noopSubmit, RunProject, RunProjectCreateInput, RunProjectUpdateInput, runProjectValidation, RunStatus, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, DeleteIcon, EditIcon } from "icons";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { RunProjectShape, shapeRunProject } from "utils/shape/models/runProject";
import { validateFormValues } from "utils/validateFormValues";
import { ScheduleUpsert } from "views/objects/schedule";
import { RunProjectFormProps, RunProjectUpsertProps } from "../types";

export const runProjectInitialValues = (
    session: Session | undefined,
    existing?: Partial<RunProject> | null | undefined,
): RunProjectShape => ({
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
});

export const transformRunProjectValues = (values: RunProjectShape, existing: RunProjectShape, isCreate: boolean) =>
    isCreate ? shapeRunProject.create(values) : shapeRunProject.update(existing, values);

const RunProjectForm = ({
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
}: RunProjectFormProps) => {
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
        endpointCreate: endpointPostRunProject,
        endpointUpdate: endpointPutRunProject,
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
                canChangeTab={false}
                canSetScheduleFor={false}
                defaultTab="RunProject"
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
};

export const RunProjectUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: RunProjectUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<RunProject, RunProjectShape>({
        ...endpointGetRunProject,
        isCreate,
        objectType: "RunProject",
        overrideObject,
        transform: (existing) => runProjectInitialValues(session, existing),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformRunProjectValues, runProjectValidation)}
        >
            {(formik) => <RunProjectForm
                disabled={!(isCreate || permissions.canUpdate)}
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
