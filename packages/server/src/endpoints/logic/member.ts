import { FindByIdInput, Member, MemberSearchInput, MemberUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsMember = {
    Query: {
        member: ApiEndpoint<FindByIdInput, FindOneResult<Member>>;
        members: ApiEndpoint<MemberSearchInput, FindManyResult<Member>>;
    },
    Mutation: {
        memberUpdate: ApiEndpoint<MemberUpdateInput, UpdateOneResult<Member>>;
    }
}

const objectType = "Member";
export const MemberEndpoints: EndpointsMember = {
    Query: {
        member: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        members: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        memberUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
