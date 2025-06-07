import { Box, Button, Card, Chip, FormControl, Grid, IconButton, InputLabel, MenuItem, Paper, Select, Stack, Typography, styled, useTheme, type Palette } from "@mui/material";
import { DUMMY_ID, DeleteType, HOURS_1_MS, ScheduleRecurrenceType, calculateOccurrences, endpointsActions, endpointsSchedule, isOfType, noopSubmit, scheduleValidation, shapeSchedule, type CalendarEvent, type CanConnect, type DeleteOneInput, type MeetingShape, type RunShape, type Schedule, type ScheduleCreateInput, type ScheduleException, type ScheduleRecurrence, type ScheduleShape, type ScheduleUpdateInput, type Session, type Success } from "@vrooli/shared";
import { addDays, format, getDay, parse, startOfWeek } from "date-fns";
import enUS from "date-fns/locale/en-US";
import { Formik, useField } from "formik";
// Removed lodash memoize import - using simple memoization instead
import { memo, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Calendar, Views, dateFnsLocalizer } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { CalendarPreviewToolbar } from "../../../components/CalendarPreviewToolbar.js";
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
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ProfileAvatar } from "../../../styles.js";
import { getDisplay, placeholderColor } from "../../../utils/display/listTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { type ScheduleForOption, type ScheduleFormProps, type ScheduleUpsertProps } from "./types.js";

export const scheduleForOptions: ScheduleForOption[] = [
    {
        iconInfo: { name: "Team", type: "Common" },
        labelKey: "Meeting",
        objectType: "Meeting",
    },
    {
        iconInfo: { name: "Play", type: "Common" },
        labelKey: "Run",
        objectType: "Run",
    },
];

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

type ScheduleForObject = CanConnect<MeetingShape | RunShape>;


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
            case "Meeting":
                return { name: "Team", type: "Common" } as const;
            default:
                return { name: "Play", type: "Common" } as const;
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
        openObject(scheduleFor, setLocation);
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
            color: palette.background.textSecondary,
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

// Simple memoization for formatRecurrence
const formatRecurrenceCache = new Map<string, string>();
const memoizedFormatRecurrence = (recurrence: ScheduleRecurrence): string => {
    const key = JSON.stringify(recurrence);
    if (!formatRecurrenceCache.has(key)) {
        formatRecurrenceCache.set(key, formatRecurrence(recurrence));
    }
    return formatRecurrenceCache.get(key)!;
};

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
                position: "relative",
                backgroundColor: palette.background.paper,
                "& .delete-button": {
                    opacity: 0,
                },
                "&:hover .delete-button": {
                    opacity: 1,
                },
            }}
            elevation={2}
        >
            {/* Summary header */}
            <Box sx={{ mb: 3, display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                <Typography variant="h6" sx={{ color: "text.primary" }}>
                    {memoizedFormatRecurrence(recurrence)}
                </Typography>
                <IconButton
                    className="delete-button"
                    edge="end"
                    size="small"
                    onClick={() => onDelete(index)}
                    sx={{
                        opacity: 0,
                        transition: "opacity 0.2s ease-in-out",
                        "&:hover": {
                            backgroundColor: palette.error.main,
                            "& svg": {
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
                            sx={{ "& .MuiSelect-select": { display: "flex", alignItems: "center", gap: 1 } }}
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
                                    {recurrence.recurrenceType.toLowerCase() + (recurrence.interval > 1 ? "s" : "")}
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

// Workaround typing issues: wrap Calendar as any to satisfy JSX
const BigCalendar: any = Calendar;

// Set up date-fns localizer for react-big-calendar
const locales = { "en-US": enUS };
const localizer = dateFnsLocalizer({ format, parse, startOfWeek, getDay, locales });

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
    pathname,
    ...props
}: ScheduleFormProps) {
    const theme = useTheme();
    const { palette, typography } = theme;
    const { t } = useTranslation();
    const styles = useScheduleFormStyles(palette);

    const [exceptionsField, , exceptionsHelpers] = useField<ScheduleException[]>("exceptions");
    const [recurrencesField, , recurrencesHelpers] = useField<ScheduleRecurrence[]>("recurrences");

    const [meetingField, , meetingHelpers] = useField<MeetingShape | null>("meeting");
    const [runField, , runHelpers] = useField<RunShape | null>("run");
    // Determine the selected object (meeting, project, or routine)
    const scheduleForObject = useMemo(() => {
        if (meetingField.value) return meetingField.value;
        if (runField.value) return runField.value;
        return null;
    }, [meetingField.value, runField.value]);
    const getScheduleForLabel = useCallback(function getScheduleForLabelCallback(scheduleFor: ScheduleForOption) {
        return t(scheduleFor.labelKey, { count: 1 });
    }, [t]);

    const [isScheduleForSearchOpen, setIsScheduleForSearchOpen] = useState(false);
    const closeScheduleForSearch = useCallback(function closeScheduleForSearchCallback(selected?: object) {
        setIsScheduleForSearchOpen(false);
        if (selected) {
            meetingHelpers.setValue(isOfType(selected, "Meeting") ? selected as MeetingShape : null);
            runHelpers.setValue(isOfType(selected, "Run") ? selected as RunShape : null);
        }
    }, [meetingHelpers, runHelpers]);
    const handleScheduleForButtonClick = useCallback(function handleScheduleForButtonClickCallback() {
        setIsScheduleForSearchOpen(true);
    }, []);

    const findScheduleForLimitTo = useMemo(function findScheduleForLimitTo() {
        return scheduleForOptions.map(option => option.objectType);
    }, []);
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
            id: DUMMY_ID,
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
        objectType: "Schedule",
        pathname,
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
    useSaveToCache({ isCreate, values, pathname });

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
                createdAt: (existing as Partial<Schedule>).createdAt ?? new Date().toISOString(),
                updatedAt: (existing as Partial<Schedule>).updatedAt ?? new Date().toISOString(),
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
                            meeting: values.meeting?.id ? {
                                __typename: "Meeting" as const,
                                id: values.meeting.id,
                            } : undefined,
                            run: values.run?.id ? {
                                __typename: "Run" as const,
                                id: values.run.id,
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

    // State for upcoming occurrence preview
    const [previewOccurrences, setPreviewOccurrences] = useState<CalendarEvent[]>([]);
    const [previewLoading, setPreviewLoading] = useState<boolean>(false);
    const [previewError, setPreviewError] = useState<string | null>(null);

    // State for mini calendar view date
    const [calendarDate, setCalendarDate] = useState<Date>(
        values.startTime ? new Date(values.startTime) : new Date(),
    );

    // Effect to recalculate occurrences for preview
    useEffect(() => {
        let isCancelled = false;
        const calculatePreview = async () => {
            // Guard against invalid dates or missing recurrences potentially causing errors
            if (!values.startTime || !values.recurrences || isNaN(new Date(values.startTime).getTime())) {
                setPreviewOccurrences([]);
                return;
            }

            setPreviewLoading(true);
            setPreviewError(null);
            const previewStartDate = new Date(values.startTime);
            const previewEndDate = addDays(previewStartDate, 60); // Preview for next 60 days

            try {
                // Cast form values to Schedule for occurrence calculation
                const occurrences = await calculateOccurrences(
                    values as any,
                    previewStartDate,
                    previewEndDate,
                );
                if (!isCancelled) {
                    setPreviewOccurrences(
                        occurrences.map(o => ({
                            __typename: "CalendarEvent",
                            id: `${values.id}|${o.start.getTime()}|${o.end.getTime()}`,
                            title: "Occurrence",
                            start: o.start,
                            end: o.end,
                            allDay: false,
                            schedule: values as any,
                        }) as CalendarEvent),
                    );
                }
            } catch (error) {
                console.error("Error calculating schedule preview:", error);
                if (!isCancelled) {
                    setPreviewError("Could not calculate preview.");
                    setPreviewOccurrences([]);
                }
            } finally {
                if (!isCancelled) {
                    setPreviewLoading(false);
                }
            }
        };

        calculatePreview();

        return () => {
            isCancelled = true; // Cleanup flag
        };
        // Dependencies: Recalculate when start time, recurrences, exceptions, or timezone change
    }, [values.startTime, values.recurrences, values.exceptions, values.timezone, values.id, values]);

    // Function to toggle exception status for a preview occurrence
    const toggleException = useCallback((occurrenceTime: Date) => {
        const occurrenceTimeStr = occurrenceTime.toISOString();
        const existingExceptions = exceptionsField.value || [];
        const existingIndex = existingExceptions.findIndex(ex => ex.originalStartTime === occurrenceTimeStr);

        if (existingIndex > -1) {
            // Exception exists, remove it
            exceptionsHelpers.setValue(existingExceptions.filter((_, i) => i !== existingIndex));
        } else {
            // Exception doesn't exist, add it
            const newException: ScheduleException = {
                __typename: "ScheduleException" as const,
                id: DUMMY_ID,
                originalStartTime: occurrenceTimeStr,
                schedule: { __typename: "Schedule" as const, id: values.id } as any,
            };
            exceptionsHelpers.setValue([...existingExceptions, newException]);
        }
    }, [exceptionsField.value, exceptionsHelpers, values.id]);

    // Memoize exception times for quick lookup
    const exceptionTimesSet = useMemo(() => {
        return new Set((exceptionsField.value || []).map(ex => ex.originalStartTime));
    }, [exceptionsField.value]);

    // Define calendar components including custom toolbar
    const calendarComponents = useMemo(() => ({
        toolbar: (props: any) => <CalendarPreviewToolbar {...props} onSelectDate={setCalendarDate} />,
    }), [setCalendarDate]);

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
                        <Grid item xs={12} md={6}>
                            <Box display="flex" flexDirection="column" maxWidth="600px" margin="auto" gap={2}>
                                <DateInput
                                    name="startTime"
                                    label="Start time"
                                    type="datetime-local"
                                    sx={{ "& .MuiInputBase-input": { fontSize: typography.caption.fontSize } }}
                                />
                                <DateInput
                                    name="endTime"
                                    label="End time"
                                    type="datetime-local"
                                    sx={{ "& .MuiInputBase-input": { fontSize: typography.caption.fontSize } }}
                                />
                                <TimezoneSelector
                                    name="timezone"
                                    label="Timezone"
                                    sx={{ "& .MuiInputBase-input": { fontSize: typography.caption.fontSize } }}
                                />
                            </Box>
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Box display="flex" flexDirection="column" maxWidth="600px" margin="auto">
                                <Typography variant="h6" sx={styles.sectionTitle}>Recurrences</Typography>
                                <Box display="flex" flexWrap="wrap" gap={1} alignItems="center" mb={2}>
                                    {recurrencesField.value.length > 0 ? recurrencesField.value.map((recurrence, index) => (
                                        <Chip key={recurrence.id} label={formatRecurrence(recurrence)} onDelete={() => removeRecurrence(index)} />
                                    )) : (
                                        <Typography variant="body2" color="text.secondary">Does not repeat</Typography>
                                    )}
                                    <Button
                                        variant="outlined"
                                        onClick={() => setRecurrenceDialogOpen(true)}
                                        size="small"
                                    >Edit recurrences</Button>
                                </Box>
                                <Typography variant="h6" sx={styles.sectionTitle}>Exceptions</Typography>
                                <Box display="flex" flexWrap="wrap" gap={1} alignItems="center" mb={2}>
                                    {exceptionsField.value.length > 0 ? exceptionsField.value.map((ex, idx) => (
                                        <Chip
                                            key={ex.id}
                                            label={new Date(ex.originalStartTime).toLocaleString()}
                                            onDelete={() => exceptionsHelpers.setValue(exceptionsField.value.filter((_, i) => i !== idx))}
                                        />
                                    )) : (
                                        <Typography variant="body2" color="text.secondary">No exceptions added</Typography>
                                    )}
                                </Box>
                            </Box>
                        </Grid>
                        <Grid item xs={12}>
                            <Box mt={4} p={2} sx={{ maxWidth: "100%", mx: "auto" }}>
                                <Typography variant="h5" sx={{ ...styles.sectionTitle, mb: 2 }}>Preview</Typography>
                                <BigCalendar
                                    localizer={localizer}
                                    events={previewOccurrences}
                                    startAccessor="start"
                                    endAccessor="end"
                                    date={calendarDate}
                                    views={[Views.MONTH, Views.WEEK, Views.DAY]}
                                    onNavigate={(date: Date) => setCalendarDate(date)}
                                    components={calendarComponents}
                                    eventPropGetter={(event: any) => {
                                        const isException = exceptionTimesSet.has(event.start.toISOString());
                                        return {
                                            style: isException
                                                ? { backgroundColor: palette.action.disabledBackground, opacity: 0.6 }
                                                : {},
                                        };
                                    }}
                                    onSelectEvent={(event: any) => toggleException(event.start)}
                                    style={{ height: 500, width: "100%" }}
                                />
                            </Box>
                        </Grid>
                        {/* Schedule-for block centered */}
                        <Grid item xs={12}>
                            <Box mt={4} p={2} sx={{ maxWidth: "600px", mx: "auto" }}>
                                {!scheduleForObject ? (
                                    <Button
                                        fullWidth
                                        variant="outlined"
                                        onClick={handleScheduleForButtonClick}
                                        sx={{
                                            borderStyle: "dashed",
                                            borderWidth: 2,
                                            borderColor: palette.divider,
                                            backgroundColor: "transparent",
                                            color: palette.text.secondary,
                                            minHeight: 120,
                                            transition: "all 0.2s ease-in-out",
                                            "&:hover": {
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
                    title={t("RecurringEvents" as any)}
                />
                <BaseForm
                    display="Dialog"
                    isLoading={isLoading}
                    maxWidth={1200}
                >
                    <Box width="100%" padding={2}>
                        <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: "medium" }}>Summary:</Typography>
                        <Typography variant="body2" sx={{ mb: 4, color: "text.secondary" }}>
                            {Array.isArray(recurrencesField.value) && recurrencesField.value.length > 0
                                ? recurrencesField.value.map(formatRecurrence).join("; ")
                                : "Does not repeat"}
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
                                transition: "all 0.2s ease-in-out",
                                "&:hover": {
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
    const [location] = useLocation();
    const pathname = location.pathname;

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
            meeting: false,
            run: false,
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
        pathname,
    }), [disabledManaged, isCreate, overrideObject, transformFn, pathname]);
    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Schedule, ScheduleShape>(managedObjectArgs);

    // Ensure Formik always receives a valid ScheduleShape for initialValues
    const formInitialValues = useMemo(() => {
        // Check for essential properties of ScheduleShape
        if (existing && typeof existing.startTime !== 'undefined' && typeof existing.endTime !== 'undefined' && typeof existing.timezone !== 'undefined') {
            return existing; // It's likely already ScheduleShape or compatible
        }
        // If 'existing' is not a full ScheduleShape, generate a default one,
        // passing the potentially partial 'existing' to retain any available data.
        return scheduleInitialValues(session, existing as Schedule);
    }, [existing, session]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={formInitialValues}
            onSubmit={noopSubmit}
            validate={(values) => validateFormValues(values, formInitialValues, isCreate, transformScheduleValues, scheduleValidation)}
        >
            {(formik) => <ScheduleForm
                canSetScheduleFor={canSetScheduleFor}
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={formInitialValues}
                handleScheduleForChange={handleScheduleForChange}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                scheduleFor={scheduleFor}
                pathname={pathname}
                {...props}
                {...formik}
            />}
        </Formik>
    );
});
