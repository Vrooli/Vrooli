import { AddIcon, DeleteIcon, DUMMY_ID, EditIcon, RunProject, runProjectValidation, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { RunProjectFormProps } from "forms/types";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { validateAndGetYupErrors } from "utils/shape/general";
import { RunProjectShape, shapeRunProject } from "utils/shape/models/runProject";
import { ScheduleUpsert } from "views/objects/schedule";

export const runProjectInitialValues = (
    session: Session | undefined,
    existing?: RunProject | null | undefined,
): RunProjectShape => ({
    __typename: "RunProject" as const,
    id: DUMMY_ID,
    ...existing,
    //TODO
});

export function transformRunProjectValues(values: RunProjectShape, existing?: RunProjectShape) {
    return existing === undefined
        ? shapeRunProject.create(values)
        : shapeRunProject.update(existing, values);
}

export const validateRunProjectValues = async (values: RunProjectShape, existing?: RunProjectShape) => {
    const transformedValues = transformRunProjectValues(values, existing);
    const validationSchema = runProjectValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const RunProjectForm = forwardRef<any, RunProjectFormProps>(({
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
                    defaultTab={CalendarPageTabOption.RunProjects}
                    display="dialog"
                    handleDelete={handleDeleteSchedule}
                    isCreate={editingSchedule === null}
                    isMutate={false}
                    onCancel={handleCloseScheduleDialog}
                    onCompleted={handleScheduleCompleted}
                    partialData={editingSchedule ?? undefined}
                    zIndex={zIndex + 1}
                />
            </LargeDialog>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(600px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {/* TODO */}
                    {/* Handle adding, updating, and removing schedule */}
                    {!scheduleField.value && (
                        <Button
                            onClick={handleAddSchedule}
                            startIcon={<AddIcon />}
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
            />
        </>
    );
});
