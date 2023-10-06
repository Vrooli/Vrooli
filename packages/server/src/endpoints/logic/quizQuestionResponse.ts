import { FindByIdInput, QuizQuestionResponse, QuizQuestionResponseSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
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
        quizQuestionResponse: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        quizQuestionResponses: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
