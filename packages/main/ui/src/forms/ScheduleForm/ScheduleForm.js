import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ScheduleRecurrenceType } from "@local/consts";
import { AddIcon, CloseIcon, DeleteIcon } from "@local/icons";
import { uuid } from "@local/uuid";
import { Box, Button, FormControl, IconButton, InputAdornment, InputLabel, MenuItem, Select, Stack, TextField, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { IntegerInput } from "../../components/inputs/IntegerInput/IntegerInput";
import { Selector } from "../../components/inputs/Selector/Selector";
import { TimezoneSelector } from "../../components/inputs/TimezoneSelector/TimezoneSelector";
import { Subheader } from "../../components/text/Subheader/Subheader";
import { BaseForm } from "../BaseForm/BaseForm";
export const ScheduleForm = forwardRef(({ display, dirty, errors, isCreate, isLoading, isOpen, onCancel, touched, values, zIndex, ...props }, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [exceptionsField, exceptionsMeta, exceptionsHelpers] = useField("exceptions");
    const [recurrencesField, recurrencesMeta, recurrencesHelpers] = useField("recurrences");
    const clearStartTime = () => {
        props.setFieldValue("startTime", "");
    };
    const clearEndTime = () => {
        props.setFieldValue("endTime", "");
    };
    const addNewRecurrence = () => {
        recurrencesHelpers.setValue([...recurrencesField.value, {
                __typename: "ScheduleRecurrence",
                id: uuid(),
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                schedule: {
                    __typename: "Schedule",
                    id: values.id,
                },
            }]);
    };
    const handleRecurrenceChange = (index, key, value) => {
        const newRecurrences = [...recurrencesField.value];
        newRecurrences[index][key] = value;
        recurrencesHelpers.setValue(newRecurrences);
    };
    const removeRecurrence = (index) => {
        recurrencesHelpers.setValue(recurrencesField.value.filter((_, idx) => idx !== index));
    };
    return (_jsxs(_Fragment, { children: [_jsx(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }, children: _jsxs(Stack, { direction: "column", spacing: 4, padding: 2, children: [_jsx(Subheader, { title: "Schedule Time Frame", help: "This section is used to define the overall time frame for the schedule.\\n\\n*Start time* and *End time* specify the beginning and the end of the period during which the schedule is active.\\n\\nThe *Timezone* is used to set the time zone for the entire schedule." }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(Field, { fullWidth: true, name: "startTime", label: "Start time (optional)", type: "datetime-local", InputProps: {
                                        endAdornment: (_jsxs(InputAdornment, { position: "end", sx: { display: "flex", alignItems: "center" }, children: [_jsx("input", { type: "hidden" }), _jsx(IconButton, { edge: "end", size: "small", onClick: clearStartTime, children: _jsx(CloseIcon, { fill: palette.background.textPrimary }) })] })),
                                    }, InputLabelProps: {
                                        shrink: true,
                                    }, as: TextField }), _jsx(Field, { fullWidth: true, name: "endTime", label: "End time (optional)", type: "datetime-local", InputProps: {
                                        endAdornment: (_jsxs(InputAdornment, { position: "end", sx: { display: "flex", alignItems: "center" }, children: [_jsx("input", { type: "hidden" }), _jsx(IconButton, { edge: "end", size: "small", onClick: clearEndTime, children: _jsx(CloseIcon, { fill: palette.background.textPrimary }) })] })),
                                    }, InputLabelProps: {
                                        shrink: true,
                                    }, as: TextField }), _jsx(TimezoneSelector, { name: "timezone", label: "Timezone" })] }), _jsx(Subheader, { title: "Recurring events", help: "Recurring events are used to set up repeated occurrences of the event in the schedule, such as daily, weekly, monthly, or yearly. *Recurrence type* determines the frequency of the repetition. *Interval* is the number of units between repetitions (e.g., every 2 weeks). Depending on the recurrence type, you may need to specify additional information such as *Day of week*, *Day of month*, or *Month of year*. Optionally, you can set an *End date* for the recurrence." }), recurrencesField.value.length ? _jsx(Box, { children: recurrencesField.value.map((recurrence, index) => (_jsx(Box, { sx: {
                                    borderRadius: 2,
                                    border: `2px solid ${palette.divider}`,
                                    marginBottom: 2,
                                    padding: 1,
                                }, children: _jsxs(Stack, { direction: "row", alignItems: "flex-start", spacing: 2, sx: { boxShadow: 6 }, children: [_jsxs(Stack, { spacing: 1, sx: { width: "100%" }, children: [_jsxs(FormControl, { fullWidth: true, children: [_jsx(InputLabel, { children: "Recurrence type" }), _jsxs(Select, { value: recurrence.recurrenceType, onChange: (e) => handleRecurrenceChange(index, "recurrenceType", e.target.value), children: [_jsx(MenuItem, { value: ScheduleRecurrenceType.Daily, children: "Daily" }), _jsx(MenuItem, { value: ScheduleRecurrenceType.Weekly, children: "Weekly" }), _jsx(MenuItem, { value: ScheduleRecurrenceType.Monthly, children: "Monthly" }), _jsx(MenuItem, { value: ScheduleRecurrenceType.Yearly, children: "Yearly" })] })] }), _jsx(TextField, { fullWidth: true, label: "Interval", type: "number", value: recurrence.interval, onChange: (e) => handleRecurrenceChange(index, "interval", parseInt(e.target.value)) }), recurrence.recurrenceType === ScheduleRecurrenceType.Weekly && (_jsx(Selector, { fullWidth: true, label: "Day of week", name: `recurrences[${index}].dayOfWeek`, options: [
                                                        { label: "Monday", value: 0 },
                                                        { label: "Tuesday", value: 1 },
                                                        { label: "Wednesday", value: 2 },
                                                        { label: "Thursday", value: 3 },
                                                        { label: "Friday", value: 4 },
                                                        { label: "Saturday", value: 5 },
                                                        { label: "Sunday", value: 6 },
                                                    ], getOptionLabel: (option) => option.label })), (recurrence.recurrenceType === ScheduleRecurrenceType.Monthly || recurrence.recurrenceType === ScheduleRecurrenceType.Yearly) && (_jsx(IntegerInput, { label: "Day of month", name: `recurrences[${index}].dayOfMonth`, min: 1, max: 31 })), recurrence.recurrenceType === ScheduleRecurrenceType.Yearly && (_jsx(Selector, { fullWidth: true, label: "Month of year", name: `recurrences[${index}].monthOfYear`, options: [
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
                                                    ], getOptionLabel: (option) => option.label })), _jsx(TextField, { fullWidth: true, label: "End date", type: "date", value: recurrence.endDate ?? "", onChange: (e) => handleRecurrenceChange(index, "endDate", e.target.value), InputLabelProps: { shrink: true } })] }), _jsx(Stack, { spacing: 1, width: 32, children: _jsx(IconButton, { edge: "end", size: "small", onClick: () => removeRecurrence(index), sx: { margin: "auto" }, children: _jsx(DeleteIcon, { fill: palette.error.light }) }) })] }) }))) }) : null, _jsx(Button, { onClick: addNewRecurrence, startIcon: _jsx(AddIcon, {}), sx: {
                                display: "flex",
                                margin: "auto",
                            }, children: "Add event" })] }) }), _jsx(GridSubmitButtons, { display: display, errors: errors, isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
});
//# sourceMappingURL=ScheduleForm.js.map