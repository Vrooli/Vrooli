import { ReminderSortBy } from "@local/consts";
import { reminderFindMany } from "../../../api/generated/endpoints/reminder_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const reminderSearchSchema = () => ({
    formLayout: searchFormLayout("SearchReminder"),
    containers: [],
    fields: [],
});
export const reminderSearchParams = () => toParams(reminderSearchSchema(), reminderFindMany, ReminderSortBy, ReminderSortBy.DueDateAsc);
//# sourceMappingURL=reminder.js.map