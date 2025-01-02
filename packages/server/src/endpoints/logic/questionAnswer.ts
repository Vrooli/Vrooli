import { FindByIdInput, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsQuestionAnswer = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<QuestionAnswer>>;
    findMany: ApiEndpoint<QuestionAnswerSearchInput, FindManyResult<QuestionAnswer>>;
    createOne: ApiEndpoint<QuestionAnswerCreateInput, CreateOneResult<QuestionAnswer>>;
    updateOne: ApiEndpoint<QuestionAnswerUpdateInput, UpdateOneResult<QuestionAnswer>>;
    acceptOne: ApiEndpoint<FindByIdInput, UpdateOneResult<QuestionAnswer>>;
}

const objectType = "QuestionAnswer";
export const questionAnswer: EndpointsQuestionAnswer = {
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
    acceptOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        throw new CustomError("000", "NotImplemented");
    },
};
