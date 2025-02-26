import { FindByIdInput, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerSearchResult, QuestionAnswerUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsQuestionAnswer = {
    findOne: ApiEndpoint<FindByIdInput, QuestionAnswer>;
    findMany: ApiEndpoint<QuestionAnswerSearchInput, QuestionAnswerSearchResult>;
    createOne: ApiEndpoint<QuestionAnswerCreateInput, QuestionAnswer>;
    updateOne: ApiEndpoint<QuestionAnswerUpdateInput, QuestionAnswer>;
    acceptOne: ApiEndpoint<FindByIdInput, QuestionAnswer>;
}

const objectType = "QuestionAnswer";
export const questionAnswer: EndpointsQuestionAnswer = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    acceptOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        throw new CustomError("000", "NotImplemented");
    },
};
