import { ReminderSortBy } from "@shared/consts";
import { reminderFindMany } from "api/generated/endpoints/reminder";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reminderSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchReminder', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const reminderSearchParams = (lng: string) => toParams(reminderSearchSchema(lng), reminderFindMany, ReminderSortBy, ReminderSortBy.DueDateAsc);