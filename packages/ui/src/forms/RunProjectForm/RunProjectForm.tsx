import { DUMMY_ID, RunProject, runProjectValidation, RunStatus, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunProjectFormProps } from "forms/types";
import { AddIcon, DeleteIcon, EditIcon } from "icons";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { validateAndGetYupErrors } from "utils/shape/general";
import { RunProjectShape, shapeRunProject } from "utils/shape/models/runProject";
import { ScheduleUpsert } from "views/objects/schedule";

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

export const validateRunProjectValues = async (values: RunProjectShape, existing: RunProjectShape, isCreate: boolean) => {
    const transformedValues = transformRunProjectValues(values, existing, isCreate);
    const validationSchema = runProjectValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const RunProjectForm = forwardRef<BaseFormRef | undefined, RunProjectFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle scheduling
    const [scheduleField, _, scheduleHelpers] = useField<Schedule | null>("schedule");
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
    const handleDeleteSchedule = () => { scheduleHelpers.setValue(null); };

    return (
        <>
            {/* Dialog to create/update schedule */}
            <ScheduleUpsert
                canChangeTab={false}
                canSetScheduleFor={false}
                defaultTab={CalendarPageTabOption.RunProjects}
                handleDelete={handleDeleteSchedule}
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                overrideObject={editingSchedule ?? { __typename: "Schedule" }}
                zIndex={zIndex + 1001}
            />
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={600}
                ref={ref}
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
                        >{"Add schedule"}</Button>
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
                                        onClick={handleDeleteSchedule}
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
