import { DUMMY_ID, endpointGetSchedule, endpointPostSchedule, endpointPutSchedule, noopSubmit, Schedule, ScheduleCreateInput, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, ScheduleUpdateInput, scheduleValidation, Session, uuid } from "@local/shared";
import { Box, Button, FormControl, IconButton, InputLabel, MenuItem, Select, Stack, TextField, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { DateInput } from "components/inputs/DateInput/DateInput";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TimezoneSelector } from "components/inputs/TimezoneSelector/TimezoneSelector";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { RelationshipButtonType } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTabs } from "hooks/useTabs";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, DeleteIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { CalendarPageTabOption, calendarTabParams } from "utils/search/objectToSearch";
import { ScheduleShape, shapeSchedule } from "utils/shape/models/schedule";
import { validateFormValues } from "utils/validateFormValues";
import { ScheduleFormProps, ScheduleUpsertProps } from "../types";

export const scheduleInitialValues = (
    session: Session | undefined,
    existing?: Schedule | null | undefined,
): ScheduleShape => ({
    __typename: "Schedule" as const,
    id: DUMMY_ID,
    startTime: new Date(),
    endTime: new Date(Date.now() + 60 * 60 * 1000), // 1 hour from now
    // Default to current timezone
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    exceptions: [],
    labels: [],
    recurrences: [],
    ...existing,
});

export const transformScheduleValues = (values: ScheduleShape, existing: ScheduleShape, isCreate: boolean) =>
    isCreate ? shapeSchedule.create(values) : shapeSchedule.update(existing, values);

const ScheduleForm = ({
    canChangeTab,
    canSetScheduleFor,
    currTab,
    disabled,
    dirty,
    existing,
    handleTabChange,
    handleUpdate,
    isCreate,
    isMutate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    tabs,
    values,
    ...props
}: ScheduleFormProps) => {
    const { palette } = useTheme();
    const display = toDisplay(isOpen);
    const { t } = useTranslation();

    const [exceptionsField, , exceptionsHelpers] = useField<ScheduleException[]>("exceptions");
    const [recurrencesField, , recurrencesHelpers] = useField<ScheduleRecurrence[]>("recurrences");

    const addNewRecurrence = () => {
        recurrencesHelpers.setValue([...recurrencesField.value, {
            __typename: "ScheduleRecurrence" as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            duration: 60 * 60 * 1000, // 1 hour
            schedule: {
                __typename: "Schedule" as const,
                id: values.id,
            } as Schedule,
        }]);
    };

    const handleRecurrenceChange = (index: number, key: keyof ScheduleRecurrence, value: any) => {
        const newRecurrences = [...recurrencesField.value];
        (newRecurrences as any)[index][key] = value;
        recurrencesHelpers.setValue(newRecurrences);
    };

    const removeRecurrence = (index: number) => {
        recurrencesHelpers.setValue(recurrencesField.value.filter((_, idx) => idx !== index));
    };

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<Schedule>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Schedule",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Schedule, ScheduleCreateInput, ScheduleUpdateInput>({
        isCreate,
        isMutate,
        endpointCreate: endpointPostSchedule,
        endpointUpdate: endpointPutSchedule,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "Schedule" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
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
                title={t(`${isCreate ? "Create" : "Update"}${currTab.tabType}` as any)}
                // Can only link to an object when creating
                below={isCreate && canChangeTab && <PageTabs
                    ariaLabel="schedule-link-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {canSetScheduleFor && <RelationshipList
                        isEditing={true}
                        limitTo={[currTab.tabType] as RelationshipButtonType[]}
                        objectType={"Schedule"}
                        sx={{ marginBottom: 4 }}
                    />}
                    <Stack direction="column" spacing={2}>
                        <Title
                            title="Schedule Time Frame"
                            help="This section is used to define the overall time frame for the schedule.\n\n*Start time* and *End time* specify the beginning and the end of the period during which the schedule is active.\n\nThe *Timezone* is used to set the time zone for the entire schedule."
                            variant="subheader"
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
                    </Stack>
                    {/* Set up recurring events */}
                    <Box>
                        <Title
                            title="Recurring events"
                            help="Recurring events are used to set up repeated occurrences of the event in the schedule, such as daily, weekly, monthly, or yearly. *Recurrence type* determines the frequency of the repetition. *Interval* is the number of units between repetitions (e.g., every 2 weeks). Depending on the recurrence type, you may need to specify additional information such as *Day of week*, *Day of month*, or *Month of year*. Optionally, you can set an *End date* for the recurrence."
                            variant="subheader"
                        />
                        {recurrencesField.value.length ? <Box>
                            {recurrencesField.value.map((recurrence, index) => (
                                <Box sx={{
                                    borderRadius: 2,
                                    marginBottom: 2,
                                    padding: 2,
                                    boxShadow: 2,
                                    background: palette.background.default,
                                }}>
                                    <Stack
                                        direction="row"
                                        alignItems="flex-start"
                                        spacing={2}
                                    >
                                        <Stack spacing={1} sx={{ width: "100%" }}>
                                            <FormControl fullWidth>
                                                <InputLabel>{"Recurrence type"}</InputLabel>
                                                <Select
                                                    value={recurrence.recurrenceType}
                                                    onChange={(e) => handleRecurrenceChange(index, "recurrenceType", e.target.value)}
                                                >
                                                    <MenuItem value={ScheduleRecurrenceType.Daily}>{"Daily"}</MenuItem>
                                                    <MenuItem value={ScheduleRecurrenceType.Weekly}>{"Weekly"}</MenuItem>
                                                    <MenuItem value={ScheduleRecurrenceType.Monthly}>{"Monthly"}</MenuItem>
                                                    <MenuItem value={ScheduleRecurrenceType.Yearly}>{"Yearly"}</MenuItem>
                                                </Select>
                                            </FormControl>
                                            <TextField
                                                fullWidth
                                                label={"Interval"}
                                                type="number"
                                                value={recurrence.interval}
                                                onChange={(e) => handleRecurrenceChange(index, "interval", parseInt(e.target.value))}
                                            />
                                            {recurrence.recurrenceType === ScheduleRecurrenceType.Weekly && (
                                                <Selector
                                                    fullWidth
                                                    label="Day of week"
                                                    name={`recurrences[${index}].dayOfWeek`}
                                                    options={[
                                                        { label: "Monday", value: 0 },
                                                        { label: "Tuesday", value: 1 },
                                                        { label: "Wednesday", value: 2 },
                                                        { label: "Thursday", value: 3 },
                                                        { label: "Friday", value: 4 },
                                                        { label: "Saturday", value: 5 },
                                                        { label: "Sunday", value: 6 },
                                                    ]}
                                                    getOptionLabel={(option) => option.label}
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
                                                    options={[
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
                                                    ]}
                                                    getOptionLabel={(option) => option.label}
                                                />
                                            )}
                                            <DateInput
                                                name={`recurrences[${index}].endDate`}
                                                label="End date"
                                                type="date"
                                            />
                                        </Stack>
                                        <Stack spacing={1} width={32}>
                                            <IconButton
                                                edge="end"
                                                size="small"
                                                onClick={() => removeRecurrence(index)}
                                                sx={{ margin: "auto" }}
                                            >
                                                <DeleteIcon fill={palette.error.light} />
                                            </IconButton>
                                        </Stack>
                                    </Stack>
                                </Box>
                            ))}
                        </Box> : null}
                        <Button
                            onClick={addNewRecurrence}
                            startIcon={<AddIcon />}
                            variant="outlined"
                            sx={{
                                display: "flex",
                                margin: "auto",
                            }}
                        >{"Add event"}</Button>
                    </Box>
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
};

const tabParams = calendarTabParams.filter(tp => tp.tabType !== "All");

export const ScheduleUpsert = ({
    canChangeTab = true,
    canSetScheduleFor = true,
    defaultTab,
    handleDelete,
    isCreate,
    isOpen,
    listId,
    overrideObject,
    ...props
}: ScheduleUpsertProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        tabs,
    } = useTabs<CalendarPageTabOption>({
        id: "schedule-tabs",
        tabParams,
        defaultTab,
        display,
    });

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Schedule, ScheduleShape>({
        ...endpointGetSchedule,
        isCreate,
        objectType: "Schedule",
        overrideObject,
        transform: (existing) => scheduleInitialValues(session, { //TODO this might cause a fetch every time a tab is changed, and we lose changed data. Need to test
            ...existing,
            // For creating, set values for linking to an object. 
            // NOTE: We can't set these values to null or undefined like you'd expect, 
            // because Formik will treat them as uncontrolled inputs and throw errors. 
            // Instead, we pretend that false is null and an empty string is undefined.
            ...(isCreate && canSetScheduleFor ? {
                focusMode: currTab.tabType === "FocusMode" ? false : "",
                meeting: currTab.tabType === "Meeting" ? false : "",
                runProject: currTab.tabType === "RunProject" ? false : "",
                runRoutine: currTab.tabType === "RunRoutine" ? false : "",
            } : {}),
        } as Schedule),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformScheduleValues, scheduleValidation)}
        >
            {(formik) => <ScheduleForm
                canSetScheduleFor={canSetScheduleFor}
                currTab={currTab}
                disabled={!(isCreate || canUpdate)}
                existing={existing}
                handleTabChange={handleTabChange}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                tabs={tabs}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
