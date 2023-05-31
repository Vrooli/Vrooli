import { FindByIdInput, Quiz, QuizCreateInput, QuizSearchInput, QuizUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsQuiz = {
    Query: {
        quiz: GQLEndpoint<FindByIdInput, FindOneResult<Quiz>>;
        quizzes: GQLEndpoint<QuizSearchInput, FindManyResult<Quiz>>;
    },
    Mutation: {
        quizCreate: GQLEndpoint<QuizCreateInput, CreateOneResult<Quiz>>;
        quizUpdate: GQLEndpoint<QuizUpdateInput, UpdateOneResult<Quiz>>;
    }
}

const objectType = "Quiz";
export const QuizEndpoints: EndpointsQuiz = {
    Query: {
        quiz: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        quizzes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        quizCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        quizUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
