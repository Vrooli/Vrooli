import { FindByIdInput, QuizQuestionResponse, QuizQuestionResponseSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../../types";

export type EndpointsQuizQuestionResponse = {
    Query: {
        quizQuestionResponse: GQLEndpoint<FindByIdInput, FindOneResult<QuizQuestionResponse>>;
        quizQuestionResponses: GQLEndpoint<QuizQuestionResponseSearchInput, FindManyResult<QuizQuestionResponse>>;
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
