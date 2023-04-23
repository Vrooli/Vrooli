import { ScheduleSortBy } from "@local/consts";
import { scheduleFindMany } from "../../../api/generated/endpoints/schedule_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const scheduleSearchSchema = () => ({
    formLayout: searchFormLayout("SearchSchedule"),
    containers: [],
    fields: [],
});
export const scheduleSearchParams = () => toParams(scheduleSearchSchema(), scheduleFindMany, ScheduleSortBy, ScheduleSortBy.DateUpdatedDesc);
//# sourceMappingURL=schedule.js.map