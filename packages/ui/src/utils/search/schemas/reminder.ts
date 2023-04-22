import { ReminderSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { reminderFindMany } from "../../../api/generated/endpoints/reminder_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reminderSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReminder"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reminderSearchParams = () => toParams(reminderSearchSchema(), reminderFindMany, ReminderSortBy, ReminderSortBy.DueDateAsc);
