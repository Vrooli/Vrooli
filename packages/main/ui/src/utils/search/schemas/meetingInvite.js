import { MeetingInviteSortBy } from "@local/consts";
import { meetingInviteFindMany } from "../../../api/generated/endpoints/meetingInvite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const meetingInviteSearchSchema = () => ({
    formLayout: searchFormLayout("SearchMeetingInvite"),
    containers: [],
    fields: [],
});
export const meetingInviteSearchParams = () => toParams(meetingInviteSearchSchema(), meetingInviteFindMany, MeetingInviteSortBy, MeetingInviteSortBy.DateCreatedDesc);
//# sourceMappingURL=meetingInvite.js.map