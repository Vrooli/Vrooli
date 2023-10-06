import { chatInvite_accept, chatInvite_create, chatInvite_decline, chatInvite_findMany, chatInvite_findOne, chatInvite_update } from "../generated";
import { ChatInviteEndpoints } from "../logic/chatInvite";
import { setupRoutes } from "./base";

export const ChatInviteRest = setupRoutes({
    "/chatInvite/:id": {
        get: [ChatInviteEndpoints.Query.chatInvite, chatInvite_findOne],
        put: [ChatInviteEndpoints.Mutation.chatInviteUpdate, chatInvite_update],
    },
    "/chatInvites": {
        get: [ChatInviteEndpoints.Query.chatInvites, chatInvite_findMany],
    },
    "/chatInvite": {
        post: [ChatInviteEndpoints.Mutation.chatInviteCreate, chatInvite_create],
    },
    "/chatInvite/:id/accept": {
        put: [ChatInviteEndpoints.Mutation.chatInviteAccept, chatInvite_accept],
    },
    "/chatInvite/:id/decline": {
        put: [ChatInviteEndpoints.Mutation.chatInviteDecline, chatInvite_decline],
    },
});
