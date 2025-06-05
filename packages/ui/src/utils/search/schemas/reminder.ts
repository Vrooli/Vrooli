import { ReminderSortBy, endpointsReminder, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
