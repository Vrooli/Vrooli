import { FindByIdInput, Question, QuestionCreateInput, QuestionSearchInput, QuestionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsQuestion = {
    Query: {
        question: ApiEndpoint<FindByIdInput, FindOneResult<Question>>;
        questions: ApiEndpoint<QuestionSearchInput, FindManyResult<Question>>;
    },
    Mutation: {
        questionCreate: ApiEndpoint<QuestionCreateInput, CreateOneResult<Question>>;
        questionUpdate: ApiEndpoint<QuestionUpdateInput, UpdateOneResult<Question>>;
    }
}

const objectType = "Question";
export const QuestionEndpoints: EndpointsQuestion = {
    Query: {
        question: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        questions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        questionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        questionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
