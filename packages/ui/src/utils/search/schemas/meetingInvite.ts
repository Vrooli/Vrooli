import { endpointGetMeetingInvite, endpointGetMeetingInvites, MeetingInviteSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingInviteSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchMeetingInvite"),
    containers: [], //TODO
    elements: [], //TODO
});

export const meetingInviteSearchParams = () => toParams(meetingInviteSearchSchema(), endpointGetMeetingInvites, endpointGetMeetingInvite, MeetingInviteSortBy, MeetingInviteSortBy.DateCreatedDesc);
