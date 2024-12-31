import { endpointsChatParticipant } from "@local/shared";
import { chatParticipant_findMany, chatParticipant_findOne, chatParticipant_update } from "../generated";
import { ChatParticipantEndpoints } from "../logic/chatParticipant";
import { setupRoutes } from "./base";

export const ChatParticipantRest = setupRoutes([
    [endpointsChatParticipant.findOne, ChatParticipantEndpoints.Query.chatParticipant, chatParticipant_findOne],
    [endpointsChatParticipant.updateOne, ChatParticipantEndpoints.Mutation.chatParticipantUpdate, chatParticipant_update],
    [endpointsChatParticipant.findMany, ChatParticipantEndpoints.Query.chatParticipants, chatParticipant_findMany],
]);
