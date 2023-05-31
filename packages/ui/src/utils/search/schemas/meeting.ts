import { meetingFindMany, MeetingSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchMeeting"),
    containers: [], //TODO
    fields: [], //TODO
});

export const meetingSearchParams = () => toParams(meetingSearchSchema(), meetingFindMany, MeetingSortBy, MeetingSortBy.AttendeesDesc);
