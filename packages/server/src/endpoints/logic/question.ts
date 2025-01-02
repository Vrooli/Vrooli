import { FindByIdInput, Question, QuestionCreateInput, QuestionSearchInput, QuestionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsQuestion = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Question>>;
    findMany: ApiEndpoint<QuestionSearchInput, FindManyResult<Question>>;
    createOne: ApiEndpoint<QuestionCreateInput, CreateOneResult<Question>>;
    updateOne: ApiEndpoint<QuestionUpdateInput, UpdateOneResult<Question>>;
}

const objectType = "Question";
export const question: EndpointsQuestion = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
