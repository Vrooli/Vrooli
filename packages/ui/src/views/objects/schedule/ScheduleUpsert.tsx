import { CanConnect, DeleteOneInput, DeleteType, DUMMY_ID, endpointsActions, endpointsSchedule, FocusModeShape, HOURS_1_MS, isOfType, LINKS, MeetingShape, noopSubmit, RunProjectShape, RunRoutineShape, Schedule, ScheduleCreateInput, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, ScheduleShape, ScheduleUpdateInput, scheduleValidation, Session, shapeSchedule, Success, uuid } from "@local/shared";
import { Box, Button, Card, Divider, FormControl, IconButton, InputLabel, MenuItem, Select, SelectChangeEvent, Stack, styled, Typography, useTheme } from "@mui/material";
import { Formik, useField } from "formik";
import { memo, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { FindObjectDialog } from "../../../components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { DateInput } from "../../../components/inputs/DateInput/DateInput.js";
import { IntegerInput } from "../../../components/inputs/IntegerInput/IntegerInput.js";
import { Selector, SelectorBase } from "../../../components/inputs/Selector/Selector.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { TimezoneSelector } from "../../../components/inputs/TimezoneSelector/TimezoneSelector.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { Title } from "../../../components/text/Title.js";
import { SessionContext } from "../../../contexts.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { AddIcon, DeleteIcon, FocusModeIcon, OpenInNewIcon, ProjectIcon, RoutineIcon, TeamIcon } from "../../../icons/common.js";
import { useLocation } from "../../../route/router.js";
import { FormSection, ProfileAvatar } from "../../../styles.js";
import { getDisplay, placeholderColor } from "../../../utils/display/listTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ScheduleFormProps, ScheduleForOption, ScheduleUpsertProps } from "./types.js";

export const scheduleForOptions: ScheduleForOption[] = [{
    Icon: TeamIcon,
    labelKey: "Meeting",
    objectType: "Meeting",
}, {
    Icon: RoutineIcon,
    labelKey: "RunRoutine",
    objectType: "RunRoutine",
}, {
    Icon: ProjectIcon,
    labelKey: "RunProject",
    objectType: "RunProject",
}, {
    Icon: FocusModeIcon,
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
    return {
        __typename: "Schedule" as const,
        id: DUMMY_ID,
        startTime: new Date(),
        endTime: new Date(Date.now() + HOURS_1_MS),
        // Default to current timezone
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        exceptions: [],
        labels: [],
        recurrences: [],
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
    const Icon = useMemo(function IconMemo() {
        switch (scheduleFor.__typename) {
            case "FocusMode":
                return FocusModeIcon;
            case "Meeting":
                return TeamIcon;
            case "RunProject":
                return ProjectIcon;
            case "RunRoutine":
                return RoutineIcon;
            default:
                return RoutineIcon;
        }
    }, [scheduleFor.__typename]);

    return (
        <ScheduleForAvatar
            alt={getDisplay(scheduleFor).title}
            isBot={true}
            profileColors={profileColors}
        >
            <Icon width="75%" height="75%" />
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
                <OpenInNewIcon />
            </Box>
        </ScheduleForCardOuter>
    );
});

const recurringEventDividerStyle = { marginTop: 2, marginBottom: 4 } as const;
const deleteButtonStyle = { margin: "auto" } as const;
const addEventButtonStyle = {
    display: "flex",
    margin: "auto",
} as const;

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
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [exceptionsField, , exceptionsHelpers] = useField<ScheduleException[]>("exceptions");
    const [recurrencesField, , recurrencesHelpers] = useField<ScheduleRecurrence[]>("recurrences");

    const [focusModeField, , focusModeHelpers] = useField<FocusModeShape | null>("focusMode");
    const [meetingField, , meetingHelpers] = useField<MeetingShape | null>("meeting");
    const [runProjectField, , runProjectHelpers] = useField<RunProjectShape | null>("runProject");
    const [runRoutineField, , runRoutineHelpers] = useField<RunRoutineShape | null>("runRoutine");
    const scheduleForObject = useMemo(function scheduleForObjectMemo() {
        switch (scheduleFor?.objectType) {
            case "FocusMode":
                return focusModeField.value;
            case "Meeting":
                return meetingField.value;
            case "RunProject":
                return runProjectField.value;
            case "RunRoutine":
                return runRoutineField.value;
            default:
                return null;
        }
    }, [focusModeField.value, meetingField.value, runProjectField.value, runRoutineField.value, scheduleFor?.objectType]);
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

    const addNewRecurrence = useCallback(function addNewRecurrenceCallback() {
        recurrencesHelpers.setValue([...recurrencesField.value, {
            __typename: "ScheduleRecurrence" as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            duration: HOURS_1_MS,
            schedule: {
                __typename: "Schedule" as const,
                id: values.id,
            } as Schedule,
        }]);
    }, [recurrencesField, recurrencesHelpers, values.id]);

    const handleRecurrenceChange = useCallback(function handleRecurrenceChangeCallback(
        index: number,
        key: keyof ScheduleRecurrence,
        value: any,
    ) {
        const newRecurrences = [...recurrencesField.value];
        (newRecurrences as any)[index][key] = value;
        recurrencesHelpers.setValue(newRecurrences);
    }, [recurrencesField, recurrencesHelpers]);

    const removeRecurrence = useCallback(function removeRecurrenceCallback(index: number) {
        recurrencesHelpers.setValue(recurrencesField.value.filter((_, idx) => idx !== index));
    }, [recurrencesField, recurrencesHelpers]);

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
            Icon: DeleteIcon,
            label: t("Delete"),
            onClick: handleDelete,
        }] : [];
    }, [handleDelete, isCreate, isMutate, t]);

    return (
        <MaybeLargeDialog
            display={display}
            id="schedule-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                options={topBarOptions}
                title={t(`Schedule${scheduleFor?.objectType ?? ""}` as const)}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    <FormSection>
                        {scheduleForObject && <ScheduleForCard scheduleFor={scheduleForObject} />}
                        {canSetScheduleFor && isCreate && <SelectorBase
                            name="scheduleFor"
                            options={scheduleForOptions}
                            getOptionLabel={getScheduleForLabel}
                            fullWidth={true}
                            inputAriaLabel="Bot style"
                            label="Bot style"
                            onChange={onScheduleForChange}
                            value={scheduleFor}
                        />}
                        {canSetScheduleFor && isCreate && <FindObjectDialog
                            find="List"
                            isOpen={isScheduleForSearchOpen}
                            limitTo={findScheduleForLimitTo}
                            handleCancel={closeScheduleForSearch}
                            handleComplete={closeScheduleForSearch as (item: object) => unknown}
                        />}
                        {canSetScheduleFor && isCreate && <Button
                            fullWidth
                            color="secondary"
                            onClick={handleScheduleForButtonClick}
                            startIcon={!scheduleForObject ? <AddIcon /> : null}
                            variant={scheduleForObject ? "outlined" : "contained"}
                        >
                            {scheduleForObject ?
                                `Change ${t(scheduleFor?.objectType ?? "", { count: 1, defaultValue: "object" })}` :
                                `Choose ${t(scheduleFor?.objectType ?? "", { count: 1, defaultValue: "object" })}`}                        </Button>}
                    </FormSection>
                    <FormSection>
                        <Title
                            title="Schedule Time Frame"
                            help="This section is used to define the overall time frame for the schedule.\n\n*Start time* and *End time* specify the beginning and the end of the period during which the schedule is active.\n\nThe *Timezone* is used to set the time zone for the entire schedule."
                            variant="subsection"
                        />
                        <DateInput
                            name="startTime"
                            label="Start time"
                            type="datetime-local"
                        />
                        <DateInput
                            name="endTime"
                            label="End time"
                            type="datetime-local"
                        />
                        <TimezoneSelector name="timezone" label="Timezone" />
                    </FormSection>
                    {/* Set up recurring events */}
                    <FormSection>
                        <Title
                            title="Recurring events"
                            help="Recurring events are used to set up repeated occurrences of the event in the schedule, such as daily, weekly, monthly, or yearly. *Recurrence type* determines the frequency of the repetition. *Interval* is the number of units between repetitions (e.g., every 2 weeks). Depending on the recurrence type, you may need to specify additional information such as *Day of week*, *Day of month*, or *Month of year*. Optionally, you can set an *End date* for the recurrence."
                            variant="subsection"
                        />
                        {recurrencesField.value.length ? <Box>
                            {recurrencesField.value.map((recurrence, index) => {
                                function onRecurrenceTypeChange(event: SelectChangeEvent<ScheduleRecurrenceType>) {
                                    handleRecurrenceChange(index, "recurrenceType", event.target.value);
                                }
                                function onIntervalChange(event: React.ChangeEvent<HTMLInputElement>) {
                                    handleRecurrenceChange(index, "interval", parseInt(event.target.value));
                                }
                                function handleRemoveRecurrence() {
                                    removeRecurrence(index);
                                }

                                return (
                                    <>
                                        <Box
                                            key={recurrence.id}
                                            display="flex"
                                            flexDirection="row"
                                            alignItems="flex-start"
                                            gap={2}
                                        >
                                            <Box
                                                display="flex"
                                                flexDirection="column"
                                                gap={2}
                                                width="-webkit-fill-available"
                                            >
                                                <FormControl fullWidth>
                                                    <InputLabel>{"Recurrence type"}</InputLabel>
                                                    <Select
                                                        value={recurrence.recurrenceType}
                                                        onChange={onRecurrenceTypeChange}
                                                    >
                                                        <MenuItem value={ScheduleRecurrenceType.Daily}>{"Daily"}</MenuItem>
                                                        <MenuItem value={ScheduleRecurrenceType.Weekly}>{"Weekly"}</MenuItem>
                                                        <MenuItem value={ScheduleRecurrenceType.Monthly}>{"Monthly"}</MenuItem>
                                                        <MenuItem value={ScheduleRecurrenceType.Yearly}>{"Yearly"}</MenuItem>
                                                    </Select>
                                                </FormControl>
                                                <TextInput
                                                    fullWidth
                                                    label={"Interval"}
                                                    type="number"
                                                    value={recurrence.interval}
                                                    onChange={onIntervalChange}
                                                />
                                                {recurrence.recurrenceType === ScheduleRecurrenceType.Weekly && (
                                                    <Selector
                                                        fullWidth
                                                        label="Day of week"
                                                        name={`recurrences[${index}].dayOfWeek`}
                                                        options={dayOfWeekOptions}
                                                        getOptionLabel={getOptionLabel}
                                                    />
                                                )}
                                                {(recurrence.recurrenceType === ScheduleRecurrenceType.Monthly || recurrence.recurrenceType === ScheduleRecurrenceType.Yearly) && (
                                                    <IntegerInput
                                                        label="Day of month"
                                                        name={`recurrences[${index}].dayOfMonth`}
                                                        min={1}
                                                        max={31}
                                                    />
                                                )}
                                                {recurrence.recurrenceType === ScheduleRecurrenceType.Yearly && (
                                                    <Selector
                                                        fullWidth
                                                        label="Month of year"
                                                        name={`recurrences[${index}].monthOfYear`}
                                                        options={monthOfYearOptions}
                                                        getOptionLabel={getOptionLabel}
                                                    />
                                                )}
                                                <DateInput
                                                    name={`recurrences[${index}].endDate`}
                                                    label="End date"
                                                    type="date"
                                                />
                                            </Box>
                                            <Stack spacing={1} width={32}>
                                                <IconButton
                                                    edge="end"
                                                    size="small"
                                                    onClick={handleRemoveRecurrence}
                                                    sx={deleteButtonStyle}
                                                >
                                                    <DeleteIcon fill={palette.error.light} />
                                                </IconButton>
                                            </Stack>
                                        </Box>
                                        <Divider sx={recurringEventDividerStyle} />
                                    </>
                                );
                            })}
                        </Box> : null}
                        <Button
                            fullWidth
                            onClick={addNewRecurrence}
                            startIcon={<AddIcon />}
                            variant="outlined"
                            sx={addEventButtonStyle}
                        >{"Add event"}</Button>
                    </FormSection>
                    {/* Set up event exceptions */}
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
}

export function ScheduleUpsert({
    canSetScheduleFor = true,
    defaultScheduleFor,
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ScheduleUpsertProps) {
    const session = useContext(SessionContext);

    const [scheduleFor, setScheduleFor] = useState<ScheduleForOption | null>(defaultScheduleFor ? scheduleForOptions.find((option) => option.objectType === defaultScheduleFor) ?? null : null);
    useEffect(function updateDefaultScheduleFor() {
        setScheduleFor(defaultScheduleFor ? scheduleForOptions.find((option) => option.objectType === defaultScheduleFor) ?? null : null);
    }, [defaultScheduleFor]);
    const handleScheduleForChange = useCallback(function handleScheduleForChangeCallback(newScheduleFor: ScheduleForOption) {
        setScheduleFor(newScheduleFor);
    }, []);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Schedule, ScheduleShape>({
        ...endpointsSchedule.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "Schedule",
        overrideObject,
        transform: (existing) => scheduleInitialValues(session, { //TODO this might cause a fetch every time a tab is changed, and we lose changed data. Need to test
            // Default scheduleFor types to false, since Formik requires inputs 
            // to be controlled and can't handle null or undefined values.
            focusMode: false,
            meeting: false,
            runProject: false,
            runRoutine: false,
            ...existing,
        } as Schedule),
    });

    async function validateValues(values: ScheduleShape) {
        return await validateFormValues(values, existing, isCreate, transformScheduleValues, scheduleValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
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
}
