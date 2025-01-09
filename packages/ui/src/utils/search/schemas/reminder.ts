import { endpointsReminder, FormSchema, ReminderSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function reminderSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchReminder"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function reminderSearchParams() {
    return toParams(reminderSearchSchema(), endpointsReminder, ReminderSortBy, ReminderSortBy.DueDateAsc);
}
