import { DUMMY_ID, RunRoutine, runRoutineValidation, RunStatus, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunRoutineFormProps } from "forms/types";
import { AddIcon, DeleteIcon, EditIcon } from "icons";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { validateAndGetYupErrors } from "utils/shape/general";
import { RunRoutineShape, shapeRunRoutine } from "utils/shape/models/runRoutine";
import { ScheduleUpsert } from "views/objects/schedule";

export const runRoutineInitialValues = (
    session: Session | undefined,
    existing?: Partial<RunRoutine> | null | undefined,
): RunRoutineShape => ({
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
});

export function transformRunRoutineValues(values: RunRoutineShape, existing?: RunRoutineShape) {
    return existing === undefined
        ? shapeRunRoutine.create(values)
        : shapeRunRoutine.update(existing, values);
}

export const validateRunRoutineValues = async (values: RunRoutineShape, existing?: RunRoutineShape) => {
    const transformedValues = transformRunRoutineValues(values, existing);
    const validationSchema = runRoutineValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const RunRoutineForm = forwardRef<BaseFormRef | undefined, RunRoutineFormProps>(({
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
            <LargeDialog
                id="schedule-dialog"
                onClose={handleCloseScheduleDialog}
                isOpen={isScheduleDialogOpen}
                titleId={""}
                zIndex={zIndex + 1}
            >
                <ScheduleUpsert
                    canChangeTab={false}
                    canSetScheduleFor={false}
                    defaultTab={CalendarPageTabOption.RunRoutines}
                    handleDelete={handleDeleteSchedule}
                    isCreate={editingSchedule === null}
                    isMutate={false}
                    onCancel={handleCloseScheduleDialog}
                    onCompleted={handleScheduleCompleted}
                    overrideObject={editingSchedule ?? { __typename: "Schedule" }}
                    zIndex={zIndex + 1001}
                />
            </LargeDialog>
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
