import { endpointsChatInvite } from "@local/shared";
import { chatInvite_accept, chatInvite_createMany, chatInvite_createOne, chatInvite_decline, chatInvite_findMany, chatInvite_findOne, chatInvite_updateMany, chatInvite_updateOne } from "../generated";
import { ChatInviteEndpoints } from "../logic/chatInvite";
import { setupRoutes } from "./base";

export const ChatInviteRest = setupRoutes([
    [endpointsChatInvite.findOne, ChatInviteEndpoints.Query.chatInvite, chatInvite_findOne],
    [endpointsChatInvite.findMany, ChatInviteEndpoints.Query.chatInvites, chatInvite_findMany],
    [endpointsChatInvite.createOne, ChatInviteEndpoints.Mutation.chatInviteCreate, chatInvite_createOne],
    [endpointsChatInvite.updateOne, ChatInviteEndpoints.Mutation.chatInviteUpdate, chatInvite_updateOne],
    [endpointsChatInvite.createMany, ChatInviteEndpoints.Mutation.chatInvitesCreate, chatInvite_createMany],
    [endpointsChatInvite.updateMany, ChatInviteEndpoints.Mutation.chatInvitesUpdate, chatInvite_updateMany],
    [endpointsChatInvite.acceptOne, ChatInviteEndpoints.Mutation.chatInviteAccept, chatInvite_accept],
    [endpointsChatInvite.declineOne, ChatInviteEndpoints.Mutation.chatInviteDecline, chatInvite_decline],
]);
