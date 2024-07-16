import { endpointGetMeeting, endpointGetMeetings, MeetingSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchMeeting"),
    containers: [], //TODO
    elements: [], //TODO
});

export const meetingSearchParams = () => toParams(meetingSearchSchema(), endpointGetMeetings, endpointGetMeeting, MeetingSortBy, MeetingSortBy.AttendeesDesc);
