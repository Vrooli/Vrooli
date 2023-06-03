import { ReminderSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reminderSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReminder"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reminderSearchParams = () => toParams(reminderSearchSchema(), "/reminders", ReminderSortBy, ReminderSortBy.DueDateAsc);
