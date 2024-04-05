import { DeleteOneInput, DeleteType, DUMMY_ID, endpointGetSchedule, endpointPostDeleteOne, endpointPostSchedule, endpointPutSchedule, noopSubmit, Schedule, ScheduleCreateInput, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, ScheduleUpdateInput, scheduleValidation, Session, Success, uuid } from "@local/shared";
import { Box, Button, FormControl, IconButton, InputLabel, MenuItem, Select, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { DateInput } from "components/inputs/DateInput/DateInput";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TimezoneSelector } from "components/inputs/TimezoneSelector/TimezoneSelector";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { RelationshipButtonType } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTabs } from "hooks/useTabs";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, DeleteIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "utils/pubsub";
import { calendarTabParams } from "utils/search/objectToSearch";
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
    display,
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
        endpointCreate: endpointPostSchedule,
        endpointUpdate: endpointPutSchedule,
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

    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
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
                options={(!isCreate && isMutate) ? [{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: handleDelete,
                }] : []}
                title={t(`${isCreate ? "Create" : "Update"}${currTab.key}` as any)}
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
                        limitTo={[currTab.key] as RelationshipButtonType[]}
                        objectType={"Schedule"}
                        sx={{ marginBottom: 4 }}
                    />}
                    <Stack direction="column" spacing={4}>
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
                                            <TextInput
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

const tabParams = calendarTabParams.filter(tp => tp.key !== "All");

export const ScheduleUpsert = ({
    canChangeTab = true,
    canSetScheduleFor = true,
    defaultTab,
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ScheduleUpsertProps) => {
    const session = useContext(SessionContext);

    const {
        currTab,
        handleTabChange,
        tabs,
    } = useTabs({
        id: "schedule-tabs",
        tabParams,
        defaultTab,
        display,
    });

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<Schedule, ScheduleShape>({
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
                focusMode: currTab.key === "FocusMode" ? false : "",
                meeting: currTab.key === "Meeting" ? false : "",
                runProject: currTab.key === "RunProject" ? false : "",
                runRoutine: currTab.key === "RunRoutine" ? false : "",
            } : {}),
        } as Schedule),
    });

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
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
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
