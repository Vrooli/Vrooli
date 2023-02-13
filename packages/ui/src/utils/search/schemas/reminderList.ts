import { ReminderListSortBy } from "@shared/consts";
import { reminderListFindMany } from "api/generated/endpoints/reminderList";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reminderListSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchReminderList', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const reminderListSearchParams = (lng: string) => toParams(reminderListSearchSchema(lng), reminderListFindMany, ReminderListSortBy, ReminderListSortBy.DateCreatedDesc);