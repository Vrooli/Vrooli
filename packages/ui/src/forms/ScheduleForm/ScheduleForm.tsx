import { Box, Button, FormControl, IconButton, InputLabel, List, ListItem, MenuItem, Select, Stack, TextField, useTheme } from "@mui/material";
import { Schedule, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType } from "@shared/consts";
import { AddIcon, DeleteIcon } from "@shared/icons";
import { uuid } from "@shared/uuid";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TimezoneSelector } from "components/inputs/TimezoneSelector/TimezoneSelector";
import { Subheader } from "components/text/Subheader/Subheader";
import { Field, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ScheduleFormProps } from "forms/types";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";

export const ScheduleForm = forwardRef<any, ScheduleFormProps>(({
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

    const [exceptionsField, exceptionsMeta, exceptionsHelpers] = useField<ScheduleException[]>('exceptions');
    const [recurrencesField, recurrencesMeta, recurrencesHelpers] = useField<ScheduleRecurrence[]>('recurrences');

    const addNewRecurrence = () => {
        console.log('setting recurrence', recurrencesField.value)
        recurrencesHelpers.setValue([...recurrencesField.value, {
            __typename: 'ScheduleRecurrence' as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            schedule: {
                __typename: 'Schedule' as const,
                id: values.id,
            } as Schedule
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
                    display: 'block',
                    maxWidth: '700px',
                    paddingBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    <Stack direction="column" spacing={2}>
                        <Field
                            fullWidth
                            name="startTime"
                            label={"Start time"}
                            type="datetime-local"
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        <Field
                            fullWidth
                            name="endTime"
                            label={"End time"}
                            type="datetime-local"
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        <TimezoneSelector name="timezone" label="Timezone" />
                    </Stack>
                    {/* Set up recurring events */}
                    <Subheader
                        title="Recurring events"
                    />
                    {recurrencesField.value.length ? <List>
                        {recurrencesField.value.map((recurrence, index) => (
                            <ListItem key={index} sx={{
                                border: `1px solid ${palette.divider}`,
                                borderRadius: 1.5,
                                marginBottom: 2,
                            }}>
                                {/* Recurrence form */}
                                <Stack direction="column" spacing={2} sx={{ display: 'block' }}>
                                    <FormControl fullWidth>
                                        <InputLabel>{"Recurrence type"}</InputLabel>
                                        <Select
                                            value={recurrence.recurrenceType}
                                            onChange={(e) => handleRecurrenceChange(index, 'recurrenceType', e.target.value)}
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
                                        onChange={(e) => handleRecurrenceChange(index, 'interval', parseInt(e.target.value))}
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
                                        onChange={(e) => handleRecurrenceChange(index, 'endDate', e.target.value)}
                                        InputLabelProps={{ shrink: true }}
                                    />
                                </Stack>
                                <Box display="flex" justifyContent="flex-end">
                                    <IconButton onClick={() => removeRecurrence(index)} color="error">
                                        <DeleteIcon fill={palette.secondary.main} />
                                    </IconButton>
                                </Box>
                            </ListItem>
                        ))}
                    </List> : null}
                    <Button
                        onClick={addNewRecurrence}
                        startIcon={<AddIcon />}
                        sx={{
                            display: 'flex',
                            margin: 'auto',
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
    )
})