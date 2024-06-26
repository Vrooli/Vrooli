import { member_findMany, member_findOne, member_update } from "../generated";
import { MemberEndpoints } from "../logic/member";
import { setupRoutes } from "./base";

export const MemberRest = setupRoutes({
    "/member/:id": {
        get: [MemberEndpoints.Query.member, member_findOne],
        put: [MemberEndpoints.Mutation.memberUpdate, member_update],
    },
    "/members": {
        get: [MemberEndpoints.Query.members, member_findMany],
    },
});
