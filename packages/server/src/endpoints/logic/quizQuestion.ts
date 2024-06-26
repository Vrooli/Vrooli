import { FindByIdInput, QuizQuestion, QuizQuestionSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../../types";

export type EndpointsQuizQuestion = {
    Query: {
        quizQuestion: GQLEndpoint<FindByIdInput, FindOneResult<QuizQuestion>>;
        quizQuestions: GQLEndpoint<QuizQuestionSearchInput, FindManyResult<QuizQuestion>>;
    },
}

const objectType = "QuizQuestion";
export const QuizQuestionEndpoints: EndpointsQuizQuestion = {
    Query: {
        quizQuestion: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        quizQuestions: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
