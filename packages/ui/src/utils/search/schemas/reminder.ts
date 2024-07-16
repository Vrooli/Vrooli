import { endpointGetReminder, endpointGetReminders, ReminderSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reminderSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReminder"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reminderSearchParams = () => toParams(reminderSearchSchema(), endpointGetReminders, endpointGetReminder, ReminderSortBy, ReminderSortBy.DueDateAsc);
