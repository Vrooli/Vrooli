import { endpointGetMeeting, endpointGetMeetings, FormSchema, MeetingSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchMeeting"),
    containers: [], //TODO
    elements: [], //TODO
});

export const meetingSearchParams = () => toParams(meetingSearchSchema(), endpointGetMeetings, endpointGetMeeting, MeetingSortBy, MeetingSortBy.AttendeesDesc);
