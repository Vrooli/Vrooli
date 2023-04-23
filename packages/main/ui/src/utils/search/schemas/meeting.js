import { MeetingSortBy } from "@local/consts";
import { meetingFindMany } from "../../../api/generated/endpoints/meeting_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const meetingSearchSchema = () => ({
    formLayout: searchFormLayout("SearchMeeting"),
    containers: [],
    fields: [],
});
export const meetingSearchParams = () => toParams(meetingSearchSchema(), meetingFindMany, MeetingSortBy, MeetingSortBy.EventStartDesc);
//# sourceMappingURL=meeting.js.map