import { endpointsMeeting, FormSchema, MeetingSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function meetingSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchMeeting"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function meetingSearchParams() {
    return toParams(meetingSearchSchema(), endpointsMeeting, MeetingSortBy, MeetingSortBy.AttendeesDesc);
}
