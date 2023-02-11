import { MeetingInviteSortBy } from "@shared/consts";
import { meetingInviteFindMany } from "api/generated/endpoints/meetingInvite";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const meetingInviteSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchMeetingInvites', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const meetingInviteSearchParams = (lng: string) => toParams(meetingInviteSearchSchema(lng), meetingInviteFindMany, MeetingInviteSortBy, MeetingInviteSortBy.DateCreatedDesc);