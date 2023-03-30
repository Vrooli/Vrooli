import { Box, Button, ListItem, Stack, TextField, Typography, useTheme } from "@mui/material";
import { Schedule } from "@shared/consts";
import { AddIcon, DeleteIcon, EditIcon, HeartFilledIcon, InvisibleIcon } from "@shared/icons";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ScheduleDialog } from "components/dialogs/ScheduleDialog/ScheduleDialog";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { Field, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { FocusModeFormProps } from "forms/types";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";

export const FocusModeForm = forwardRef<any, FocusModeFormProps>(({
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
    const [scheduleField, scheduleMeta, scheduleHelpers] = useField<Schedule | null>('schedule');
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    const handleAddSchedule = () => { setIsScheduleDialogOpen(true) };
    const handleUpdateSchedule = () => {
        setEditingSchedule(scheduleField.value);
        setIsScheduleDialogOpen(true);
    };
    const handleCloseScheduleDialog = () => { setIsScheduleDialogOpen(false) };
    const handleScheduleCreated = (created: Schedule) => {
        scheduleHelpers.setValue(created);
        setIsScheduleDialogOpen(false);
    };
    const handleScheduleUpdated = (updated: Schedule) => {
        scheduleHelpers.setValue(updated);
        setIsScheduleDialogOpen(false);
    };
    const handleDeleteSchedule = () => { scheduleHelpers.setValue(null) };

    return (
        <>
            {/* Dialog to create/update schedule */}
            <ScheduleDialog
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onClose={handleCloseScheduleDialog}
                onCreated={handleScheduleCreated}
                onUpdated={handleScheduleUpdated}
                zIndex={zIndex + 1}
            />
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    width: 'min(600px, 100vw)',
                    paddingBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    <Stack direction="column" spacing={2}>
                        <Field
                            fullWidth
                            name="name"
                            label={t('Name')}
                            as={TextField}
                        />
                        <Field
                            fullWidth
                            name="description"
                            label={t('Description')}
                            as={TextField}
                        />
                    </Stack>
                    {/* Handle adding, updating, and removing schedule */}
                    {!scheduleField.value && (
                        <Button
                            onClick={handleAddSchedule}
                            startIcon={<AddIcon />}
                            sx={{
                                display: 'flex',
                                margin: 'auto',
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
                                        width: '-webkit-fill-available',
                                        display: 'grid',
                                        pointerEvents: 'none',
                                    }}
                                >
                                    {/* TODO */}
                                </Stack>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    sx={{
                                        pointerEvents: 'none',
                                        justifyContent: 'center',
                                        alignItems: 'start',
                                    }}
                                >
                                    {/* Edit */}
                                    <Box
                                        component="a"
                                        onClick={handleUpdateSchedule}
                                        sx={{
                                            display: 'flex',
                                            justifyContent: 'center',
                                            alignItems: 'center',
                                            cursor: 'pointer',
                                            pointerEvents: 'all',
                                            paddingBottom: '4px',
                                        }}>
                                        <EditIcon fill={palette.secondary.main} />
                                    </Box>
                                    {/* Delete */}
                                    <Box
                                        component="a"
                                        onClick={handleDeleteSchedule}
                                        sx={{
                                            display: 'flex',
                                            justifyContent: 'center',
                                            alignItems: 'center',
                                            cursor: 'pointer',
                                            pointerEvents: 'all',
                                            paddingBottom: '4px',
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
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <HeartFilledIcon fill={palette.background.textPrimary} />
                        <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('TopicsFavorite')}</Typography>
                        <HelpButton markdown={t('TopicsFavoriteHelp')} />
                    </Stack>
                    <TagSelector />
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <InvisibleIcon fill={palette.background.textPrimary} />
                        <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('TopicsHidden')}</Typography>
                        <HelpButton markdown={t('TopicsHiddenHelp')} />
                    </Stack>
                    <TagSelector />
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
    )
})