import { FindByIdInput, Meeting, MeetingCreateInput, MeetingSearchInput, MeetingUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsMeeting = {
    Query: {
        meeting: GQLEndpoint<FindByIdInput, FindOneResult<Meeting>>;
        meetings: GQLEndpoint<MeetingSearchInput, FindManyResult<Meeting>>;
    },
    Mutation: {
        meetingCreate: GQLEndpoint<MeetingCreateInput, CreateOneResult<Meeting>>;
        meetingUpdate: GQLEndpoint<MeetingUpdateInput, UpdateOneResult<Meeting>>;
    }
}

const objectType = "Meeting";
export const MeetingEndpoints: EndpointsMeeting = {
    Query: {
        meeting: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        meetings: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        meetingCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        meetingUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
