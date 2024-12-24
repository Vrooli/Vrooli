import { FindByIdInput, Label, LabelCreateInput, LabelSearchInput, LabelUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsLabel = {
    Query: {
        label: ApiEndpoint<FindByIdInput, FindOneResult<Label>>;
        labels: ApiEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        labelCreate: ApiEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        labelUpdate: ApiEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
}

const objectType = "Label";
export const LabelEndpoints: EndpointsLabel = {
    Query: {
        label: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        labels: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        labelCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        labelUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
