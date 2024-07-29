import { endpointGetReminder, endpointGetReminders, FormSchema, ReminderSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reminderSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReminder"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reminderSearchParams = () => toParams(reminderSearchSchema(), endpointGetReminders, endpointGetReminder, ReminderSortBy, ReminderSortBy.DueDateAsc);
