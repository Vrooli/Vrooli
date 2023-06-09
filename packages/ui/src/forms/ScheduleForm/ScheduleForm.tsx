import { AddIcon, CloseIcon, DeleteIcon, DUMMY_ID, Schedule, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, scheduleValidation, Session, uuid } from "@local/shared";
import { Box, Button, FormControl, IconButton, InputAdornment, InputLabel, MenuItem, Select, Stack, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TimezoneSelector } from "components/inputs/TimezoneSelector/TimezoneSelector";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { Title } from "components/text/Title/Title";
import { Field, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ScheduleFormProps } from "forms/types";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ScheduleShape, shapeSchedule } from "utils/shape/models/schedule";

export const scheduleInitialValues = (
    session: Session | undefined,
    existing?: Schedule | null | undefined,
): ScheduleShape => ({
    __typename: "Schedule" as const,
    id: DUMMY_ID,
    startTime: null,
    endTime: null,
    // Default to current timezone
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    exceptions: [],
    labels: [],
    recurrences: [],
    ...existing,
});

export const transformScheduleValues = (values: ScheduleShape, existing?: ScheduleShape) => {
    return existing === undefined
        ? shapeSchedule.create(values)
        : shapeSchedule.update(existing, values);
};

export const validateScheduleValues = async (values: ScheduleShape, existing?: ScheduleShape) => {
    const transformedValues = transformScheduleValues(values, existing);
    const validationSchema = scheduleValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};


export const ScheduleForm = forwardRef<any, ScheduleFormProps>(({
    canSetScheduleFor,
    display,
    dirty,
    errors,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    touched,
    values,
    zIndex,
    ...props
}, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [exceptionsField, exceptionsMeta, exceptionsHelpers] = useField<ScheduleException[]>("exceptions");
    const [recurrencesField, recurrencesMeta, recurrencesHelpers] = useField<ScheduleRecurrence[]>("recurrences");

    const clearStartTime = () => {
        props.setFieldValue("startTime", "");
    };

    const clearEndTime = () => {
        props.setFieldValue("endTime", "");
    };

    const addNewRecurrence = () => {
        recurrencesHelpers.setValue([...recurrencesField.value, {
            __typename: "ScheduleRecurrence" as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
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

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {canSetScheduleFor && <RelationshipList
                        isEditing={true}
                        objectType={"Schedule"}
                        zIndex={zIndex}
                        sx={{ marginBottom: 4 }}
                    />}
                    <Title
                        title="Schedule Time Frame"
                        help="This section is used to define the overall time frame for the schedule.\n\n*Start time* and *End time* specify the beginning and the end of the period during which the schedule is active.\n\nThe *Timezone* is used to set the time zone for the entire schedule."
                        variant="subheader"
                    />
                    <Stack direction="column" spacing={2}>
                        <Field
                            fullWidth
                            name="startTime"
                            label={"Start time (optional)"}
                            type="datetime-local"
                            InputProps={{
                                endAdornment: (
                                    <InputAdornment position="end" sx={{ display: "flex", alignItems: "center" }}>
                                        <input type="hidden" />
                                        <IconButton edge="end" size="small" onClick={clearStartTime}>
                                            <CloseIcon fill={palette.background.textPrimary} />
                                        </IconButton>
                                    </InputAdornment>
                                ),
                            }}
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        <Field
                            fullWidth
                            name="endTime"
                            label={"End time (optional)"}
                            type="datetime-local"
                            InputProps={{
                                endAdornment: (
                                    <InputAdornment position="end" sx={{ display: "flex", alignItems: "center" }}>
                                        <input type="hidden" />
                                        <IconButton edge="end" size="small" onClick={clearEndTime}>
                                            <CloseIcon fill={palette.background.textPrimary} />
                                        </IconButton>
                                    </InputAdornment>
                                ),
                            }}
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        <TimezoneSelector name="timezone" label="Timezone" />
                    </Stack>
                    {/* Set up recurring events */}
                    <Title
                        title="Recurring events"
                        help="Recurring events are used to set up repeated occurrences of the event in the schedule, such as daily, weekly, monthly, or yearly. *Recurrence type* determines the frequency of the repetition. *Interval* is the number of units between repetitions (e.g., every 2 weeks). Depending on the recurrence type, you may need to specify additional information such as *Day of week*, *Day of month*, or *Month of year*. Optionally, you can set an *End date* for the recurrence."
                        variant="subheader"
                    />
                    {recurrencesField.value.length ? <Box>
                        {recurrencesField.value.map((recurrence, index) => (
                            <Box sx={{
                                borderRadius: 2,
                                border: `2px solid ${palette.divider}`,
                                marginBottom: 2,
                                padding: 1,
                            }}>
                                <Stack
                                    direction="row"
                                    alignItems="flex-start"
                                    spacing={2}
                                    sx={{ boxShadow: 6 }}
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
                                        <TextField
                                            fullWidth
                                            label={"End date"}
                                            type="date"
                                            value={recurrence.endDate ?? ""}
                                            onChange={(e) => handleRecurrenceChange(index, "endDate", e.target.value)}
                                            InputLabelProps={{ shrink: true }}
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
                        sx={{
                            display: "flex",
                            margin: "auto",
                        }}
                    >{"Add event"}</Button>
                    {/* Set up event exceptions */}
                    {/* TODO */}
                </Stack>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={errors as any}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});
