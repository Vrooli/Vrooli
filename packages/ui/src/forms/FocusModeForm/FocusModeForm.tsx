import { AddIcon, DeleteIcon, DUMMY_ID, EditIcon, FocusMode, focusModeValidation, HeartFilledIcon, InvisibleIcon, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { Title } from "components/text/Title/Title";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { FocusModeFormProps } from "forms/types";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { validateAndGetYupErrors } from "utils/shape/general";
import { FocusModeShape, shapeFocusMode } from "utils/shape/models/focusMode";
import { ScheduleUpsert } from "views/objects/schedule";

export const focusModeInitialValues = (
    session: Session | undefined,
    existing?: FocusMode | null | undefined,
): FocusModeShape => ({
    __typename: "FocusMode" as const,
    id: DUMMY_ID,
    description: "",
    name: "",
    reminderList: {
        __typename: "ReminderList" as const,
        id: DUMMY_ID,
        reminders: [],
    },
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
        resources: [],
    },
    filters: [],
    schedule: null,
    ...existing,
});

export function transformFocusModeValues(values: FocusModeShape, existing?: FocusModeShape) {
    return existing === undefined
        ? shapeFocusMode.create(values)
        : shapeFocusMode.update(existing, values);
}

export const validateFocusModeValues = async (values: FocusModeShape, existing?: FocusModeShape) => {
    const transformedValues = transformFocusModeValues(values, existing);
    const validationSchema = focusModeValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const FocusModeForm = forwardRef<BaseFormRef | undefined, FocusModeFormProps>(({
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
                    defaultTab={CalendarPageTabOption.FocusModes}
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
                display={display}
                isLoading={isLoading}
                maxWidth={600}
                ref={ref}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    <Stack direction="column" spacing={2}>
                        <Field
                            fullWidth
                            name="name"
                            label={t("Name")}
                            as={TextField}
                        />
                        <Field
                            fullWidth
                            name="description"
                            label={t("Description")}
                            as={TextField}
                        />
                    </Stack>
                    {/* Handle adding, updating, and removing schedule */}
                    {!scheduleField.value && (
                        <Button
                            onClick={handleAddSchedule}
                            startIcon={<AddIcon />}
                            sx={{
                                display: "flex",
                                margin: "auto",
                            }}
                            variant="outlined"
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
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <Title
                        Icon={HeartFilledIcon}
                        title={t("TopicsFavorite")}
                        help={t("TopicsFavoriteHelp")}
                        variant="subheader"
                    />
                    <TagSelector
                        name="favorites"
                        zIndex={zIndex}
                    />
                    <Title
                        Icon={InvisibleIcon}
                        title={t("TopicsHidden")}
                        help={t("TopicsHiddenHelp")}
                        variant="subheader"
                    />
                    <TagSelector
                        name="hidden"
                        zIndex={zIndex}
                    />
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
