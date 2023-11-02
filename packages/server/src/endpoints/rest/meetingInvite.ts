import { meetingInvite_accept, meetingInvite_createMany, meetingInvite_createOne, meetingInvite_decline, meetingInvite_findMany, meetingInvite_findOne, meetingInvite_updateMany, meetingInvite_updateOne } from "../generated";
import { MeetingInviteEndpoints } from "../logic/meetingInvite";
import { setupRoutes } from "./base";

export const MeetingInviteRest = setupRoutes({
    "/meetingInvite/:id": {
        get: [MeetingInviteEndpoints.Query.meetingInvite, meetingInvite_findOne],
        put: [MeetingInviteEndpoints.Mutation.meetingInviteUpdate, meetingInvite_updateOne],
    },
    "/meetingInvites": {
        get: [MeetingInviteEndpoints.Query.meetingInvites, meetingInvite_findMany],
        post: [MeetingInviteEndpoints.Mutation.meetingInvitesCreate, meetingInvite_createMany],
        put: [MeetingInviteEndpoints.Mutation.meetingInvitesUpdate, meetingInvite_updateMany],
    },
    "/meetingInvite": {
        post: [MeetingInviteEndpoints.Mutation.meetingInviteCreate, meetingInvite_createOne],
    },
    "/meetingInvite/:id/accept": {
        put: [MeetingInviteEndpoints.Mutation.meetingInviteAccept, meetingInvite_accept],
    },
    "/meetingInvite/:id/decline": {
        put: [MeetingInviteEndpoints.Mutation.meetingInviteDecline, meetingInvite_decline],
    },
});
