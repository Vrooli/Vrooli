import { CanConnect, DeleteOneInput, DeleteType, DUMMY_ID, endpointsActions, endpointsSchedule, FocusModeShape, HOURS_1_MS, isOfType, LINKS, MeetingShape, noopSubmit, RunProjectShape, RunRoutineShape, Schedule, ScheduleCreateInput, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, ScheduleShape, ScheduleUpdateInput, scheduleValidation, Session, shapeSchedule, Success, uuid } from "@local/shared";
import { Box, Button, Card, Chip, FormControl, Grid, IconButton, InputLabel, MenuItem, Palette, Paper, Select, Stack, styled, Typography, useTheme } from "@mui/material";
import { Formik, useField } from "formik";
import memoize from 'lodash/memoize';
import { memo, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { FindObjectDialog } from "../../../components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { DateInput } from "../../../components/inputs/DateInput/DateInput.js";
import { IntegerInput } from "../../../components/inputs/IntegerInput/IntegerInput.js";
import { Selector } from "../../../components/inputs/Selector/Selector.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { TimezoneSelector } from "../../../components/inputs/TimezoneSelector/TimezoneSelector.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ProfileAvatar } from "../../../styles.js";
import { getDisplay, placeholderColor } from "../../../utils/display/listTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ScheduleFormProps, ScheduleForOption, ScheduleUpsertProps } from "./types.js";

export const scheduleForOptions: ScheduleForOption[] = [{
    iconInfo: { name: "Team", type: "Common" },
    labelKey: "Meeting",
    objectType: "Meeting",
}, {
    iconInfo: { name: "Routine", type: "Routine" },
    labelKey: "RunRoutine",
    objectType: "RunRoutine",
}, {
    iconInfo: { name: "Project", type: "Common" },
    labelKey: "RunProject",
    objectType: "RunProject",
}, {
    iconInfo: { name: "FocusMode", type: "Common" },
    labelKey: "FocusMode",
    objectType: "FocusMode",
}];

const dayOfWeekOptions = [
    { label: "Monday", value: 0 },
    { label: "Tuesday", value: 1 },
    { label: "Wednesday", value: 2 },
    { label: "Thursday", value: 3 },
    { label: "Friday", value: 4 },
    { label: "Saturday", value: 5 },
    { label: "Sunday", value: 6 },
] as const;

const monthOfYearOptions = [
    { label: "January", value: 0 },
    { label: "February", value: 1 },
    { label: "March", value: 2 },
    { label: "April", value: 3 },
    { label: "May", value: 4 },
    { label: "June", value: 5 },
    { label: "July", value: 6 },
    { label: "August", value: 7 },
    { label: "September", value: 8 },
    { label: "October", value: 9 },
    { label: "November", value: 10 },
    { label: "December", value: 11 },
] as const;

function getOptionLabel(option: { label: string }) {
    return option.label;
}

export function scheduleInitialValues(
    session: Session | undefined,
    existing?: Schedule | null | undefined,
): ScheduleShape {
    const base = {
        __typename: "Schedule" as const,
        id: DUMMY_ID,
        startTime: new Date(),
        endTime: new Date(Date.now() + HOURS_1_MS),
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        exceptions: [],
        labels: [],
        recurrences: [],
    };

    if (!existing) return base;
    return {
        ...base,
        ...existing,
    };
}

export function transformScheduleValues(values: ScheduleShape, existing: ScheduleShape, isCreate: boolean) {
    return isCreate ? shapeSchedule.create(values) : shapeSchedule.update(existing, values);
}

type ScheduleForObject = CanConnect<FocusModeShape | MeetingShape | RunProjectShape | RunRoutineShape>;


const ScheduleForCardOuter = styled(Card)(({ theme }) => ({
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    padding: 0,
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    borderRadius: "8px",
}));

export const ScheduleForAvatar = styled(ProfileAvatar)(() => ({
    width: 72,
    height: 72,
    borderRadius: "0px",
}));

const ScheduleForCardAvatar = memo(function ScheduleForCardAvatarMemo({ scheduleFor }: { scheduleFor: ScheduleForObject }) {
    const profileColors = useMemo(() => placeholderColor(), []);
    const iconInfo = useMemo(function iconInfoMemo() {
        switch (scheduleFor.__typename) {
            case "FocusMode":
                return { name: "FocusMode", type: "Common" } as const;
            case "Meeting":
                return { name: "Team", type: "Common" } as const;
            case "RunProject":
                return { name: "Project", type: "Common" } as const;
            case "RunRoutine":
                return { name: "Routine", type: "Routine" } as const;
            default:
                return { name: "Routine", type: "Routine" } as const;
        }
    }, [scheduleFor.__typename]);

    return (
        <ScheduleForAvatar
            alt={getDisplay(scheduleFor).title}
            isBot={true}
            profileColors={profileColors}
        >
            <Icon
                decorative
                info={iconInfo}
            />
        </ScheduleForAvatar>
    );
});

const scheduleForCardOpenIconBoxStyle = {
    display: "grid",
    marginLeft: "auto",
    paddingRight: 1,
} as const;

type ScheduleForCardProps = {
    scheduleFor: ScheduleForObject;
}

const ScheduleForCard = memo(function ScheduleForCardMemo({
    scheduleFor,
}: ScheduleForCardProps) {
    const [, setLocation] = useLocation();

    const handleCardClick = useCallback(function handleCardClickCallback() {
        // Focus modes should direct you to the focus modes settings page
        if (scheduleFor.__typename === "FocusMode") {
            setLocation(LINKS.SettingsFocusModes);
        } else {
            openObject(scheduleFor, setLocation);
        }
    }, [scheduleFor, setLocation]);

    return (
        <ScheduleForCardOuter onClick={handleCardClick}>
            <ScheduleForCardAvatar scheduleFor={scheduleFor} />
            <Box p={1}>
                <Typography variant="h6">{getDisplay(scheduleFor).title}</Typography>
                {getDisplay(scheduleFor).subtitle && <Typography variant="body2">{getDisplay(scheduleFor).subtitle}</Typography>}
            </Box>
            <Box sx={scheduleForCardOpenIconBoxStyle}>
                <IconCommon
                    decorative
                    name="OpenInNew"
                />
            </Box>
        </ScheduleForCardOuter>
    );
});

const dialogSx = {
    paper: {
        width: "min(100%, 1200px)",
    },
};

const AddButton = styled(Button)(({ theme }) => ({
    alignSelf: "center",
    backgroundColor: theme.palette.background.paper,
    borderRadius: 0,
    color: theme.palette.background.textSecondary,
}));

function useScheduleFormStyles(palette: Palette) {
    return useMemo(() => ({
        sectionTitle: {
            marginBottom: 1,
        },
        deleteIconButton: {
            position: "absolute",
            top: 8,
            right: 8,
            zIndex: 1,
        },
        dividerStyle: {
            display: { xs: "flex", lg: "none" },
            marginBottom: { xs: 2, lg: 0 },
            marginTop: { xs: 2, lg: 0 },
        },
        recurrencePaper: {
            background: palette.background.paper,
            borderBottom: `1px solid ${palette.divider}`,
            borderRadius: 0,
            p: 1.5,
            position: "relative",
        },
        addEventButton: {
            alignSelf: "center",
            backgroundColor: palette.background.paper,
            borderRadius: 0,
            color: palette.background.textSecondary,
        },
    }), [palette]);
}

/**
 * Generate a human-readable summary label for a recurrence rule.
 */
function formatRecurrence(r: ScheduleRecurrence): string {
    const { recurrenceType, interval, dayOfWeek, dayOfMonth, month } = r;
    switch (recurrenceType) {
        case ScheduleRecurrenceType.Daily:
            return interval > 1 ? `Every ${interval} days` : "Daily";
        case ScheduleRecurrenceType.Weekly: {
            const d = dayOfWeekOptions.find(o => o.value === dayOfWeek)?.label;
            const base = interval > 1 ? `Every ${interval} weeks` : "Weekly";
            return d ? `${base} on ${d}` : base;
        }
        case ScheduleRecurrenceType.Monthly:
            return interval > 1
                ? `Every ${interval} months on day ${dayOfMonth}`
                : `Monthly on day ${dayOfMonth}`;
        case ScheduleRecurrenceType.Yearly: {
            const m = monthOfYearOptions.find(o => o.value === month)?.label;
            const base = interval > 1 ? `Every ${interval} years` : "Yearly";
            return m ? `${base} on ${m} ${dayOfMonth}` : base;
        }
        default:
            return "";
    }
}

// Ensure proper memoization of the recurrence formatter
const memoizedFormatRecurrence = memoize(formatRecurrence);

// Create a memoized component for the recurrence item to prevent unnecessary re-renders
const RecurrenceItem = memo(function RecurrenceItem({
    recurrence,
    index,
    onDelete,
    onChange,
    palette,
}: {
    recurrence: ScheduleRecurrence;
    index: number;
    onDelete: (index: number) => void;
    onChange: (index: number, key: keyof ScheduleRecurrence, value: any) => void;
    palette: Palette;
}) {
    // Use the memoized formatRecurrence function directly
    return (
        <Paper
            sx={{
                p: 3,
                position: 'relative',
                backgroundColor: palette.background.paper,
                '& .delete-button': {
                    opacity: 0,
                },
                '&:hover .delete-button': {
                    opacity: 1,
                },
            }}
            elevation={2}
        >
            {/* Summary header */}
            <Box sx={{ mb: 3, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography variant="h6" sx={{ color: 'text.primary' }}>
                    {memoizedFormatRecurrence(recurrence)}
                </Typography>
                <IconButton
                    className="delete-button"
                    edge="end"
                    size="small"
                    onClick={() => onDelete(index)}
                    sx={{
                        opacity: 0,
                        transition: 'opacity 0.2s ease-in-out',
                        '&:hover': {
                            backgroundColor: palette.error.main,
                            '& svg': {
                                fill: palette.common.white,
                            },
                        },
                    }}
                >
                    <IconCommon decorative fill={palette.error.light} name="Delete" />
                </IconButton>
            </Box>

            {/* Main form content */}
            <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                    <FormControl fullWidth>
                        <InputLabel>Repeats</InputLabel>
                        <Select
                            value={recurrence.recurrenceType}
                            onChange={(event) => onChange(index, "recurrenceType", event.target.value)}
                            sx={{ '& .MuiSelect-select': { display: 'flex', alignItems: 'center', gap: 1 } }}
                        >
                            <MenuItem value={ScheduleRecurrenceType.Daily}>
                                <IconCommon decorative name="Schedule" type="Common" style={{ fontSize: 20 }} />
                                Daily
                            </MenuItem>
                            <MenuItem value={ScheduleRecurrenceType.Weekly}>
                                <IconCommon decorative name="Schedule" type="Common" style={{ fontSize: 20 }} />
                                Weekly
                            </MenuItem>
                            <MenuItem value={ScheduleRecurrenceType.Monthly}>
                                <IconCommon decorative name="Schedule" type="Common" style={{ fontSize: 20 }} />
                                Monthly
                            </MenuItem>
                            <MenuItem value={ScheduleRecurrenceType.Yearly}>
                                <IconCommon decorative name="Schedule" type="Common" style={{ fontSize: 20 }} />
                                Yearly
                            </MenuItem>
                        </Select>
                    </FormControl>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <TextInput
                        fullWidth
                        label="Repeat every"
                        type="number"
                        value={recurrence.interval}
                        onChange={(event) => onChange(index, "interval", parseInt(event.target.value))}
                        InputProps={{
                            endAdornment: (
                                <Typography variant="body2" color="text.secondary" sx={{ ml: 1 }}>
                                    {recurrence.recurrenceType.toLowerCase() + (recurrence.interval > 1 ? 's' : '')}
                                </Typography>
                            ),
                        }}
                    />
                </Grid>

                {/* Conditional fields based on recurrence type */}
                {recurrence.recurrenceType === ScheduleRecurrenceType.Weekly && (
                    <Grid item xs={12}>
                        <Selector
                            fullWidth
                            label="Repeats on"
                            name={`recurrences[${index}].dayOfWeek`}
                            options={dayOfWeekOptions}
                            getOptionLabel={getOptionLabel}
                        />
                    </Grid>
                )}
                {(recurrence.recurrenceType === ScheduleRecurrenceType.Monthly || recurrence.recurrenceType === ScheduleRecurrenceType.Yearly) && (
                    <Grid item xs={12} sm={6}>
                        <IntegerInput
                            label="Day of month"
                            name={`recurrences[${index}].dayOfMonth`}
                            min={1}
                            max={31}
                        />
                    </Grid>
                )}
                {recurrence.recurrenceType === ScheduleRecurrenceType.Yearly && (
                    <Grid item xs={12} sm={6}>
                        <Selector
                            fullWidth
                            label="Month"
                            name={`recurrences[${index}].monthOfYear`}
                            options={monthOfYearOptions}
                            getOptionLabel={getOptionLabel}
                        />
                    </Grid>
                )}

                {/* End date */}
                <Grid item xs={12}>
                    <DateInput
                        name={`recurrences[${index}].endDate`}
                        label="Ends on"
                        type="date"
                    />
                </Grid>
            </Grid>
        </Paper>
    );
});

function ScheduleForm({
    canSetScheduleFor,
    disabled,
    display,
    existing,
    handleScheduleForChange,
    handleUpdate,
    isCreate,
    isMutate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    scheduleFor,
    values,
    ...props
}: ScheduleFormProps) {
    const theme = useTheme();
    const { palette, typography } = theme;
    const { t } = useTranslation();
    const styles = useScheduleFormStyles(palette);

    const [exceptionsField, , exceptionsHelpers] = useField<ScheduleException[]>("exceptions");
    const [recurrencesField, , recurrencesHelpers] = useField<ScheduleRecurrence[]>("recurrences");

    const [focusModeField, , focusModeHelpers] = useField<FocusModeShape | null>("focusMode");
    const [meetingField, , meetingHelpers] = useField<MeetingShape | null>("meeting");
    const [runProjectField, , runProjectHelpers] = useField<RunProjectShape | null>("runProject");
    const [runRoutineField, , runRoutineHelpers] = useField<RunRoutineShape | null>("runRoutine");
    // Determine the selected object (focusMode, meeting, project, or routine)
    const scheduleForObject = useMemo(() => {
        if (focusModeField.value) return focusModeField.value;
        if (meetingField.value) return meetingField.value;
        if (runProjectField.value) return runProjectField.value;
        if (runRoutineField.value) return runRoutineField.value;
        return null;
    }, [focusModeField.value, meetingField.value, runProjectField.value, runRoutineField.value]);
    const getScheduleForLabel = useCallback(function getScheduleForLabelCallback(scheduleFor: ScheduleForOption) {
        return t(scheduleFor.labelKey, { count: 1 });
    }, [t]);

    const [isScheduleForSearchOpen, setIsScheduleForSearchOpen] = useState(false);
    const closeScheduleForSearch = useCallback(function closeScheduleForSearchCallback(selected?: object) {
        setIsScheduleForSearchOpen(false);
        if (selected) {
            focusModeHelpers.setValue(isOfType(selected, "FocusMode") ? selected as FocusModeShape : null);
            meetingHelpers.setValue(isOfType(selected, "Meeting") ? selected as MeetingShape : null);
            runProjectHelpers.setValue(isOfType(selected, "RunProject") ? selected as RunProjectShape : null);
            runRoutineHelpers.setValue(isOfType(selected, "RunRoutine") ? selected as RunRoutineShape : null);
        }
    }, [focusModeHelpers, meetingHelpers, runProjectHelpers, runRoutineHelpers]);
    const handleScheduleForButtonClick = useCallback(function handleScheduleForButtonClickCallback() {
        setIsScheduleForSearchOpen(true);
    }, []);

    const findScheduleForLimitTo = useMemo(function findScheduleForLimitTo() {
        return scheduleFor?.objectType ? [scheduleFor.objectType] : [];
    }, [scheduleFor?.objectType]);
    const onScheduleForChange = useCallback(function onScheduleForChangeCallback(selected: ScheduleForOption) {
        handleScheduleForChange(selected);
        setIsScheduleForSearchOpen(true);
    }, [handleScheduleForChange]);

    const handleRecurrenceChange = useCallback(function handleRecurrenceChangeCallback(
        index: number,
        key: keyof ScheduleRecurrence,
        value: any,
    ) {
        if (!Array.isArray(recurrencesField.value)) return;

        const newRecurrences = [...recurrencesField.value];
        if (newRecurrences[index]) {
            (newRecurrences[index] as any)[key] = value;
            recurrencesHelpers.setValue(newRecurrences);
        }
    }, [recurrencesField.value, recurrencesHelpers]);

    const removeRecurrence = useCallback(function removeRecurrenceCallback(index: number) {
        const currentRecurrences = recurrencesField.value || [];
        if (!Array.isArray(currentRecurrences)) {
            recurrencesHelpers.setValue([]);
            return;
        }
        recurrencesHelpers.setValue(currentRecurrences.filter((_, idx) => idx !== index));
    }, [recurrencesField.value, recurrencesHelpers]);

    const addNewRecurrence = useCallback(function addNewRecurrenceCallback() {
        const currentRecurrences = recurrencesField.value || [];
        if (!Array.isArray(currentRecurrences)) {
            recurrencesHelpers.setValue([]);
            return;
        }
        recurrencesHelpers.setValue([...currentRecurrences, {
            __typename: "ScheduleRecurrence" as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            duration: 60,
            schedule: {
                __typename: "Schedule" as const,
                id: values.id,
            } as Schedule,
        }]);
    }, [recurrencesField.value, recurrencesHelpers, values.id]);

    const { handleCancel, handleCreated, handleCompleted, handleDeleted } = useUpsertActions<Schedule>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Schedule",
        ...props,
    });
    const {
        fetch,
        fetchCreate,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Schedule, ScheduleCreateInput, ScheduleUpdateInput>({
        isCreate,
        isMutate,
        endpointCreate: endpointsSchedule.createOne,
        endpointUpdate: endpointsSchedule.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Schedule" });

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (isMutate) {
            fetchLazyWrapper<ScheduleCreateInput | ScheduleUpdateInput, Schedule>({
                fetch,
                inputs: transformScheduleValues(values, existing, isCreate),
                onSuccess: (data) => { handleCompleted(data); },
                onCompleted: () => { props.setSubmitting(false); },
            });
        } else {
            onCompleted?.({
                ...values,
                created_at: (existing as Partial<Schedule>).created_at ?? new Date().toISOString(),
                updated_at: (existing as Partial<Schedule>).updated_at ?? new Date().toISOString(),
            } as Schedule);
        }
    }, [disabled, existing, fetch, handleCompleted, isCreate, isMutate, onCompleted, props, values]);

    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const handleDelete = useCallback(() => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteMutation,
            inputs: { id: values.id, objectType: DeleteType.Schedule },
            successCondition: (data) => data.success,
            successMessage: () => ({
                messageKey: "ObjectDeleted",
                messageVariables: { objectName: t("Schedule", { count: 1 }) },
                buttonKey: "Undo",
                buttonClicked: () => {
                    fetchLazyWrapper<ScheduleCreateInput, Schedule>({
                        fetch: fetchCreate,
                        inputs: transformScheduleValues({
                            ...values,
                            // Make sure not to set any extra fields, 
                            // so the relationship is treated as a "Connect" instead of a "Create"
                            focusMode: values.focusMode?.id ? {
                                __typename: "FocusMode" as const,
                                id: values.focusMode.id,
                            } : undefined,
                            meeting: values.meeting?.id ? {
                                __typename: "Meeting" as const,
                                id: values.meeting.id,
                            } : undefined,
                            runProject: values.runProject?.id ? {
                                __typename: "RunProject" as const,
                                id: values.runProject.id,
                            } : undefined,
                            runRoutine: values.runRoutine?.id ? {
                                __typename: "RunRoutine" as const,
                                id: values.runRoutine.id,
                            } : undefined,
                        }, values, true) as ScheduleCreateInput,
                        successCondition: (data) => !!data.id,
                        onSuccess: (data) => { handleCreated(data); },
                    });
                },
            }),
            onSuccess: () => {
                handleDeleted(values as Schedule);
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
        });
    }, [deleteMutation, fetchCreate, handleCreated, handleDeleted, t, values]);

    const isLoading = useMemo(() => isCreateLoading || isDeleteLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isDeleteLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const topBarOptions = useMemo(function topBarOptionsMemo() {
        return (!isCreate && isMutate) ? [{
            iconInfo: { name: "Delete", type: "Common" } as const,
            label: t("Delete"),
            onClick: handleDelete,
        }] : [];
    }, [handleDelete, isCreate, isMutate, t]);

    // state for recurrences editor dialog
    const [recurrenceDialogOpen, setRecurrenceDialogOpen] = useState(false);

    return (
        <MaybeLargeDialog
            display={display}
            id="schedule-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={dialogSx}
        >
            <TopBar
                display={display}
                onClose={onClose}
                options={topBarOptions}
                title={t("Schedule", { count: 1 })}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={1200}
            >
                <Box width="100%" padding={2}>
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <Box display="flex" flexDirection="column" gap={4} maxWidth="600px" margin="auto">
                                <Typography variant="h5" sx={styles.sectionTitle}>Time Frame</Typography>
                                <Grid container spacing={2}>
                                    <Grid item xs={12} md={6}>
                                        <DateInput
                                            name="startTime"
                                            label="Start time"
                                            type="datetime-local"
                                            sx={{ '& .MuiInputBase-input': { fontSize: typography.caption.fontSize } }}
                                        />
                                    </Grid>
                                    <Grid item xs={12} md={6}>
                                        <DateInput
                                            name="endTime"
                                            label="End time"
                                            type="datetime-local"
                                            sx={{ '& .MuiInputBase-input': { fontSize: typography.caption.fontSize } }}
                                        />
                                    </Grid>
                                    <Grid item xs={12}>
                                        <TimezoneSelector
                                            name="timezone"
                                            label="Timezone"
                                            sx={{ '& .MuiInputBase-input': { fontSize: typography.caption.fontSize } }}
                                        />
                                    </Grid>
                                </Grid>
                            </Box>
                        </Grid>
                        {/* Recurring events & Exceptions side-by-side */}
                        <Grid item xs={12}>
                            <Box mt={4} p={2} sx={{ maxWidth: '600px', mx: 'auto' }}>
                                <Grid container spacing={2}>
                                    <Grid item xs={12} md={6}>
                                        <Typography variant="h5" sx={styles.sectionTitle}>Recurring events</Typography>
                                        <Box display="flex" flexWrap="wrap" gap={1} alignItems="center">
                                            {recurrencesField.value.length > 0 ? recurrencesField.value.map((recurrence, index) => (
                                                <Chip key={recurrence.id} label={formatRecurrence(recurrence)} onDelete={() => removeRecurrence(index)} />
                                            )) : (
                                                <Typography variant="body2" color="text.secondary">Does not repeat</Typography>
                                            )}
                                            <Button variant="outlined" onClick={() => setRecurrenceDialogOpen(true)}>Edit recurrences</Button>
                                        </Box>
                                    </Grid>
                                    <Grid item xs={12} md={6}>
                                        <Typography variant="h5" sx={styles.sectionTitle}>Exceptions</Typography>
                                        {exceptionsField.value.length > 0 ? exceptionsField.value.map((ex, idx) => (
                                            <Box key={ex.id} display="flex" alignItems="center" gap={1} mb={1}>
                                                <DateInput name={`exceptions[${idx}].date`} label="Exception date" type="date" />
                                                <IconButton size="small" onClick={() => exceptionsHelpers.setValue(exceptionsField.value.filter((_, i) => i !== idx))}>
                                                    <IconCommon decorative fill="error.main" name="Delete" />
                                                </IconButton>
                                            </Box>
                                        )) : (
                                            <Typography variant="body2" color="text.secondary">No exceptions</Typography>
                                        )}
                                        <Button
                                            variant="outlined"
                                            onClick={() => exceptionsHelpers.setValue([
                                                ...exceptionsField.value,
                                                ({
                                                    __typename: "ScheduleException" as const,
                                                    id: uuid(),
                                                    originalStartTime: new Date().toISOString(),
                                                    schedule: { __typename: "Schedule" as const, id: values.id } as any,
                                                } as ScheduleException)
                                            ])}
                                        >Add exception</Button>
                                    </Grid>
                                </Grid>
                            </Box>
                        </Grid>
                        {/* Schedule-for block centered */}
                        <Grid item xs={12}>
                            <Box mt={4} p={2} sx={{ maxWidth: '600px', mx: 'auto' }}>
                                {!scheduleForObject ? (
                                    <Button
                                        fullWidth
                                        variant="outlined"
                                        onClick={handleScheduleForButtonClick}
                                        sx={{
                                            borderStyle: 'dashed',
                                            borderWidth: 2,
                                            borderColor: palette.divider,
                                            backgroundColor: 'transparent',
                                            color: palette.text.secondary,
                                            minHeight: 120,
                                            transition: 'all 0.2s ease-in-out',
                                            '&:hover': {
                                                backgroundColor: palette.action.hover,
                                            },
                                        }}
                                    >
                                        Add object
                                    </Button>
                                ) : (
                                    <>
                                        <ScheduleForCard scheduleFor={scheduleForObject} />
                                        <Button fullWidth onClick={handleScheduleForButtonClick} sx={{ mt: 2 }}>
                                            Change object
                                        </Button>
                                    </>
                                )}
                            </Box>
                        </Grid>
                        <Grid item xs={12} sx={{ mt: 4 }}>
                            <FindObjectDialog
                                find="List"
                                isOpen={isScheduleForSearchOpen}
                                limitTo={findScheduleForLimitTo}
                                handleCancel={closeScheduleForSearch}
                                handleComplete={closeScheduleForSearch as (item: object) => unknown}
                            />
                        </Grid>
                    </Grid>
                </Box>
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
            {/* recurrences editor dialog */}
            <MaybeLargeDialog
                display="Dialog"
                id="schedule-recurrences-dialog"
                isOpen={recurrenceDialogOpen}
                onClose={() => setRecurrenceDialogOpen(false)}
                sxs={dialogSx}
            >
                <TopBar
                    display="Dialog"
                    onClose={() => setRecurrenceDialogOpen(false)}
                    title={t("RecurringEvents")}
                />
                <BaseForm
                    display="Dialog"
                    isLoading={isLoading}
                    maxWidth={1200}
                >
                    <Box width="100%" padding={2}>
                        <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 'medium' }}>Summary:</Typography>
                        <Typography variant="body2" sx={{ mb: 4, color: 'text.secondary' }}>
                            {Array.isArray(recurrencesField.value) && recurrencesField.value.length > 0
                                ? recurrencesField.value.map(formatRecurrence).join('; ')
                                : 'Does not repeat'}
                        </Typography>
                        {Array.isArray(recurrencesField.value) && recurrencesField.value.length > 0 && (
                            <Box borderRadius={2} overflow="overlay" sx={{ mb: 2 }}>
                                <Stack spacing={2}>
                                    {recurrencesField.value.map((recurrence, index) => (
                                        <RecurrenceItem
                                            key={recurrence.id}
                                            recurrence={recurrence}
                                            index={index}
                                            onDelete={removeRecurrence}
                                            onChange={handleRecurrenceChange}
                                            palette={palette}
                                        />
                                    ))}
                                </Stack>
                            </Box>
                        )}
                        <AddButton
                            fullWidth
                            onClick={addNewRecurrence}
                            startIcon={<IconCommon decorative name="Add" />}
                            sx={{
                                transition: 'all 0.2s ease-in-out',
                                '&:hover': {
                                    backgroundColor: palette.action.hover,
                                },
                            }}
                        >
                            Add event
                        </AddButton>
                    </Box>
                </BaseForm>
                <BottomActionsButtons
                    display="Dialog"
                    hideButtons={disabled}
                    isCreate={isCreate}
                    loading={isLoading}
                    onCancel={() => setRecurrenceDialogOpen(false)}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={() => setRecurrenceDialogOpen(false)}
                />
            </MaybeLargeDialog>
        </MaybeLargeDialog>
    );
}

// Memoized ScheduleUpsert to avoid re-renders when props haven't changed
export const ScheduleUpsert = memo(function ScheduleUpsert({
    canSetScheduleFor = true,
    defaultScheduleFor,
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ScheduleUpsertProps) {
    const session = useContext(SessionContext);
    // Do not render or manage form when dialog is hidden
    if (display === "Dialog" && !isOpen) {
        return null;
    }

    const [scheduleFor, setScheduleFor] = useState<ScheduleForOption | null>(defaultScheduleFor ? scheduleForOptions.find((option) => option.objectType === defaultScheduleFor) ?? null : null);
    useEffect(function updateDefaultScheduleFor() {
        setScheduleFor(defaultScheduleFor ? scheduleForOptions.find((option) => option.objectType === defaultScheduleFor) ?? null : null);
    }, [defaultScheduleFor]);
    const handleScheduleForChange = useCallback(function handleScheduleForChangeCallback(newScheduleFor: ScheduleForOption) {
        setScheduleFor(newScheduleFor);
    }, []);

    // Memoize transform to avoid re-creating inline function each render and prevent endless rerenders
    const transformFn = useCallback((data: Partial<Schedule>) => {
        // Ensure controlled fields for form before applying existing data
        const baseline: Schedule = {
            focusMode: false,
            meeting: false,
            runProject: false,
            runRoutine: false,
            ...data,
        } as Schedule;
        return scheduleInitialValues(session, baseline);
    }, [session]);
    // Disable managed hook entirely for create mode to avoid overriding form data each fetch cycle
    const disabledManaged = isCreate || (display === "Dialog" && isOpen !== true);
    const managedObjectArgs = useMemo(() => ({
        ...endpointsSchedule.findOne,
        disabled: disabledManaged,
        isCreate,
        objectType: "Schedule" as const,
        overrideObject: !isCreate ? overrideObject : undefined,
        transform: transformFn,
    }), [disabledManaged, isCreate, overrideObject, transformFn]);
    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Schedule, ScheduleShape>(managedObjectArgs);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={(values) => validateFormValues(values, existing, isCreate, transformScheduleValues, scheduleValidation)}
        >
            {(formik) => <ScheduleForm
                canSetScheduleFor={canSetScheduleFor}
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
                handleScheduleForChange={handleScheduleForChange}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                scheduleFor={scheduleFor}
                {...props}
                {...formik}
            />}
        </Formik>
    );
});
