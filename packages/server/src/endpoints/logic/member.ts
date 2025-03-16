import { FindByIdInput, Member, MemberSearchInput, MemberSearchResult, MemberUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsMember = {
    findOne: ApiEndpoint<FindByIdInput, Member>;
    findMany: ApiEndpoint<MemberSearchInput, MemberSearchResult>;
    updateOne: ApiEndpoint<MemberUpdateInput, Member>;
}

const objectType = "Member";
export const member: EndpointsMember = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
