import { UserScheduleSortBy } from "@shared/consts";
import { userScheduleFindMany } from "api/generated/endpoints/userSchedule";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const userScheduleSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchUserSchedules', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const userScheduleSearchParams = (lng: string) => toParams(userScheduleSearchSchema(lng), userScheduleFindMany, UserScheduleSortBy, UserScheduleSortBy.EventStartAsc)