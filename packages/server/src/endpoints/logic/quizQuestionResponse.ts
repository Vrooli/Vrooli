import { FindByIdInput, QuizQuestionResponse, QuizQuestionResponseSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";

export type EndpointsQuizQuestionResponse = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<QuizQuestionResponse>>;
    findMany: ApiEndpoint<QuizQuestionResponseSearchInput, FindManyResult<QuizQuestionResponse>>;
}

const objectType = "QuizQuestionResponse";
export const quizQuestionResponse: EndpointsQuizQuestionResponse = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
