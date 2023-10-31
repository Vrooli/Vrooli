import { chatInvite_accept, chatInvite_createMany, chatInvite_createOne, chatInvite_decline, chatInvite_findMany, chatInvite_findOne, chatInvite_updateMany, chatInvite_updateOne } from "../generated";
import { ChatInviteEndpoints } from "../logic/chatInvite";
import { setupRoutes } from "./base";

export const ChatInviteRest = setupRoutes({
    "/chatInvite/:id": {
        get: [ChatInviteEndpoints.Query.chatInvite, chatInvite_findOne],
        put: [ChatInviteEndpoints.Mutation.chatInviteUpdate, chatInvite_updateOne],
    },
    "/chatInvites": {
        get: [ChatInviteEndpoints.Query.chatInvites, chatInvite_findMany],
        post: [ChatInviteEndpoints.Mutation.chatInvitesCreate, chatInvite_createMany],
        put: [ChatInviteEndpoints.Mutation.chatInvitesUpdate, chatInvite_updateMany],
    },
    "/chatInvite": {
        post: [ChatInviteEndpoints.Mutation.chatInviteCreate, chatInvite_createOne],
    },
    "/chatInvite/:id/accept": {
        put: [ChatInviteEndpoints.Mutation.chatInviteAccept, chatInvite_accept],
    },
    "/chatInvite/:id/decline": {
        put: [ChatInviteEndpoints.Mutation.chatInviteDecline, chatInvite_decline],
    },
});
