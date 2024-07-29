import { endpointGetMeetingInvite, endpointGetMeetingInvites, FormSchema, MeetingInviteSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingInviteSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchMeetingInvite"),
    containers: [], //TODO
    elements: [], //TODO
});

export const meetingInviteSearchParams = () => toParams(meetingInviteSearchSchema(), endpointGetMeetingInvites, endpointGetMeetingInvite, MeetingInviteSortBy, MeetingInviteSortBy.DateCreatedDesc);
