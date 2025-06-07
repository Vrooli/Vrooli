import { type ChatParticipant, type ChatParticipantSearchInput, type ChatParticipantSearchResult, type ChatParticipantUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsChatParticipant = {
    findOne: ApiEndpoint<FindByIdInput, ChatParticipant>;
    findMany: ApiEndpoint<ChatParticipantSearchInput, ChatParticipantSearchResult>;
    updateOne: ApiEndpoint<ChatParticipantUpdateInput, ChatParticipant>;
}

const objectType = "ChatParticipant";
export const chatParticipant: EndpointsChatParticipant = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
