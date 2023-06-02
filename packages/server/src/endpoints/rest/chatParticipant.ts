import { chatParticipant_findMany, chatParticipant_findOne, chatParticipant_update } from "@local/shared";
import { ChatParticipantEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ChatParticipantRest = setupRoutes({
    "/chatParticipant/:id": {
        get: [ChatParticipantEndpoints.Query.chatParticipant, chatParticipant_findOne],
        put: [ChatParticipantEndpoints.Mutation.chatParticipantUpdate, chatParticipant_update],
    },
    "/chatParticipants": {
        get: [ChatParticipantEndpoints.Query.chatParticipants, chatParticipant_findMany],
    },
});
