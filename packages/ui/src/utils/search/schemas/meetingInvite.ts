import { endpointsMeetingInvite, FormSchema, MeetingInviteSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function meetingInviteSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchMeetingInvite"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function meetingInviteSearchParams() {
    return toParams(meetingInviteSearchSchema(), endpointsMeetingInvite, MeetingInviteSortBy, MeetingInviteSortBy.DateCreatedDesc);
}
