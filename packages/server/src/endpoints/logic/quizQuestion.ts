import { FindByIdInput, QuizQuestion, QuizQuestionSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions";
import { rateLimit } from "../../middleware";
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
        quizQuestion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        quizQuestions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
