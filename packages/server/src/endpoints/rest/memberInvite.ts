import { memberInvite_accept, memberInvite_createMany, memberInvite_createOne, memberInvite_decline, memberInvite_findMany, memberInvite_findOne, memberInvite_updateMany, memberInvite_updateOne } from "../generated";
import { MemberInviteEndpoints } from "../logic/memberInvite";
import { setupRoutes } from "./base";

export const MemberInviteRest = setupRoutes({
    "/memberInvite/:id": {
        get: [MemberInviteEndpoints.Query.memberInvite, memberInvite_findOne],
        put: [MemberInviteEndpoints.Mutation.memberInviteUpdate, memberInvite_updateOne],
    },
    "/memberInvites": {
        get: [MemberInviteEndpoints.Query.memberInvites, memberInvite_findMany],
        post: [MemberInviteEndpoints.Mutation.memberInvitesCreate, memberInvite_createMany],
        put: [MemberInviteEndpoints.Mutation.memberInvitesUpdate, memberInvite_updateMany],
    },
    "/memberInvite": {
        post: [MemberInviteEndpoints.Mutation.memberInviteCreate, memberInvite_createOne],
    },
    "/memberInvite/:id/accept": {
        put: [MemberInviteEndpoints.Mutation.memberInviteAccept, memberInvite_accept],
    },
    "/memberInvite/:id/decline": {
        put: [MemberInviteEndpoints.Mutation.memberInviteDecline, memberInvite_decline],
    },
});
