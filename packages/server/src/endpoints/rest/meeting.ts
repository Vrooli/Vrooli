import { endpointsMeeting } from "@local/shared";
import { meeting_create, meeting_findMany, meeting_findOne, meeting_update } from "../generated";
import { MeetingEndpoints } from "../logic/meeting";
import { setupRoutes } from "./base";

export const MeetingRest = setupRoutes([
    [endpointsMeeting.findOne, MeetingEndpoints.Query.meeting, meeting_findOne],
    [endpointsMeeting.findMany, MeetingEndpoints.Query.meetings, meeting_findMany],
    [endpointsMeeting.createOne, MeetingEndpoints.Mutation.meetingCreate, meeting_create],
    [endpointsMeeting.updateOne, MeetingEndpoints.Mutation.meetingUpdate, meeting_update],
]);
