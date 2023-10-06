import { meetingInvite_accept, meetingInvite_create, meetingInvite_decline, meetingInvite_findMany, meetingInvite_findOne, meetingInvite_update } from "../generated";
import { MeetingInviteEndpoints } from "../logic/meetingInvite";
import { setupRoutes } from "./base";

export const MeetingInviteRest = setupRoutes({
    "/meetingInvite/:id": {
        get: [MeetingInviteEndpoints.Query.meetingInvite, meetingInvite_findOne],
        put: [MeetingInviteEndpoints.Mutation.meetingInviteUpdate, meetingInvite_update],
    },
    "/meetingInvites": {
        get: [MeetingInviteEndpoints.Query.meetingInvites, meetingInvite_findMany],
    },
    "/meetingInvite": {
        post: [MeetingInviteEndpoints.Mutation.meetingInviteCreate, meetingInvite_create],
    },
    "/meetingInvite/:id/accept": {
        put: [MeetingInviteEndpoints.Mutation.meetingInviteAccept, meetingInvite_accept],
    },
    "/meetingInvite/:id/decline": {
        put: [MeetingInviteEndpoints.Mutation.meetingInviteDecline, meetingInvite_decline],
    },
});
