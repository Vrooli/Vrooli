import { MeetingSortBy } from "@shared/consts";
import { meetingFindMany } from "api/generated/endpoints/meeting";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchMeeting', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const meetingSearchParams = (lng: string) => toParams(meetingSearchSchema(lng), meetingFindMany, MeetingSortBy, MeetingSortBy.EventStartDesc);