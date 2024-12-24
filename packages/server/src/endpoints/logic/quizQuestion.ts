import { FindByIdInput, QuizQuestion, QuizQuestionSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";

export type EndpointsQuizQuestion = {
    Query: {
        quizQuestion: ApiEndpoint<FindByIdInput, FindOneResult<QuizQuestion>>;
        quizQuestions: ApiEndpoint<QuizQuestionSearchInput, FindManyResult<QuizQuestion>>;
    },
}

const objectType = "QuizQuestion";
export const QuizQuestionEndpoints: EndpointsQuizQuestion = {
    Query: {
        quizQuestion: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        quizQuestions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
