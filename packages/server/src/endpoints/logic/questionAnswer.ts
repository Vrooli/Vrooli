import { FindByIdInput, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsQuestionAnswer = {
    Query: {
        questionAnswer: GQLEndpoint<FindByIdInput, FindOneResult<QuestionAnswer>>;
        questionAnswers: GQLEndpoint<QuestionAnswerSearchInput, FindManyResult<QuestionAnswer>>;
    },
    Mutation: {
        questionAnswerCreate: GQLEndpoint<QuestionAnswerCreateInput, CreateOneResult<QuestionAnswer>>;
        questionAnswerUpdate: GQLEndpoint<QuestionAnswerUpdateInput, UpdateOneResult<QuestionAnswer>>;
        questionAnswerMarkAsAccepted: GQLEndpoint<FindByIdInput, UpdateOneResult<QuestionAnswer>>;
    }
}

const objectType = "QuestionAnswer";
export const QuestionAnswerEndpoints: EndpointsQuestionAnswer = {
    Query: {
        questionAnswer: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        questionAnswers: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        questionAnswerCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        questionAnswerUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        questionAnswerMarkAsAccepted: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            throw new CustomError("000", "NotImplemented");
        },
    },
};
