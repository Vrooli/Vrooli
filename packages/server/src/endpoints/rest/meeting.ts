import { meeting_create, meeting_findMany, meeting_findOne, meeting_update } from "@local/shared";
import { MeetingEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const MeetingRest = setupRoutes({
    "/meeting/:id": {
        get: [MeetingEndpoints.Query.meeting, meeting_findOne],
        put: [MeetingEndpoints.Mutation.meetingUpdate, meeting_update],
    },
    "/meetings": {
        get: [MeetingEndpoints.Query.meetings, meeting_findMany],
    },
    "/meeting": {
        post: [MeetingEndpoints.Mutation.meetingCreate, meeting_create],
    },
});
