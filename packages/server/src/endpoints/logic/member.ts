import { FindByIdInput, Member, MemberSearchInput, MemberUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsMember = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Member>>;
    findMany: ApiEndpoint<MemberSearchInput, FindManyResult<Member>>;
    updateOne: ApiEndpoint<MemberUpdateInput, UpdateOneResult<Member>>;
}

const objectType = "Member";
export const member: EndpointsMember = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
