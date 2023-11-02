import { DUMMY_ID, RunRoutine, runRoutineValidation, RunStatus, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
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

export const transformRunRoutineValues = (values: RunRoutineShape, existing: RunRoutineShape, isCreate: boolean) =>
    isCreate ? shapeRunRoutine.create(values) : shapeRunRoutine.update(existing, values);

export const validateRunRoutineValues = async (values: RunRoutineShape, existing: RunRoutineShape, isCreate: boolean) => {
    const transformedValues = transformRunRoutineValues(values, existing, isCreate);
    const validationSchema = runRoutineValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
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
                defaultTab={CalendarPageTabOption.RunRoutine}
                handleDelete={handleDeleteSchedule}
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
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
