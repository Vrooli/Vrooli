import { DUMMY_ID, FocusMode, focusModeValidation, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, TextField, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { Title } from "components/text/Title/Title";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { FocusModeFormProps } from "forms/types";
import { AddIcon, DeleteIcon, EditIcon, HeartFilledIcon, InvisibleIcon } from "icons";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { validateAndGetYupErrors } from "utils/shape/general";
import { FocusModeShape, shapeFocusMode } from "utils/shape/models/focusMode";
import { ScheduleUpsert } from "views/objects/schedule";

export const focusModeInitialValues = (
    session: Session | undefined,
    existing?: Partial<FocusMode> | null | undefined,
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

export const transformFocusModeValues = (values: FocusModeShape, existing: FocusModeShape, isCreate: boolean) =>
    isCreate ? shapeFocusMode.create(values) : shapeFocusMode.update(existing, values);

export const validateFocusModeValues = async (values: FocusModeShape, existing: FocusModeShape, isCreate: boolean) => {
    const transformedValues = transformFocusModeValues(values, existing, isCreate);
    const validationSchema = focusModeValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
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
                defaultTab={CalendarPageTabOption.FocusMode}
                handleDelete={handleDeleteSchedule}
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                overrideObject={editingSchedule ?? { __typename: "Schedule" }}
            />
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
                        parent={{ __typename: "FocusMode", id: values.id }}
                    />
                    <Title
                        Icon={HeartFilledIcon}
                        title={t("TopicsFavorite")}
                        help={t("TopicsFavoriteHelp")}
                        variant="subheader"
                    />
                    <TagSelector name="favorites" />
                    <Title
                        Icon={InvisibleIcon}
                        title={t("TopicsHidden")}
                        help={t("TopicsHiddenHelp")}
                        variant="subheader"
                    />
                    <TagSelector name="hidden" />
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
