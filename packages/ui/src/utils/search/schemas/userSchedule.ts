import { UserScheduleSortBy } from "@shared/consts";
import { userScheduleFindMany } from "api/generated/endpoints/userSchedule";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const userScheduleSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchUserSchedule'),
    containers: [], //TODO
    fields: [], //TODO
})

export const userScheduleSearchParams = () => toParams(userScheduleSearchSchema(), userScheduleFindMany, UserScheduleSortBy, UserScheduleSortBy.EventStartAsc)