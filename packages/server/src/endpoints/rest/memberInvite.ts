import { memberInvite_accept, memberInvite_create, memberInvite_decline, memberInvite_findMany, memberInvite_findOne, memberInvite_update } from "../generated";
import { MemberInviteEndpoints } from "../logic/memberInvite";
import { setupRoutes } from "./base";

export const MemberInviteRest = setupRoutes({
    "/memberInvite/:id": {
        get: [MemberInviteEndpoints.Query.memberInvite, memberInvite_findOne],
        put: [MemberInviteEndpoints.Mutation.memberInviteUpdate, memberInvite_update],
    },
    "/memberInvites": {
        get: [MemberInviteEndpoints.Query.memberInvites, memberInvite_findMany],
    },
    "/memberInvite": {
        post: [MemberInviteEndpoints.Mutation.memberInviteCreate, memberInvite_create],
    },
    "/memberInvite/:id/accept": {
        put: [MemberInviteEndpoints.Mutation.memberInviteAccept, memberInvite_accept],
    },
    "/memberInvite/:id/decline": {
        put: [MemberInviteEndpoints.Mutation.memberInviteDecline, memberInvite_decline],
    },
});
