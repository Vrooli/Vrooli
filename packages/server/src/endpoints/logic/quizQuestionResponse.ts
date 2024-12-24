import { FindByIdInput, QuizQuestionResponse, QuizQuestionResponseSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";

export type EndpointsQuizQuestionResponse = {
    Query: {
        quizQuestionResponse: ApiEndpoint<FindByIdInput, FindOneResult<QuizQuestionResponse>>;
        quizQuestionResponses: ApiEndpoint<QuizQuestionResponseSearchInput, FindManyResult<QuizQuestionResponse>>;
    },
}

const objectType = "QuizQuestionResponse";
export const QuizQuestionResponseEndpoints: EndpointsQuizQuestionResponse = {
    Query: {
        quizQuestionResponse: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        quizQuestionResponses: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
