import { FindByIdInput, QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsQuizAttempt = {
    Query: {
        quizAttempt: GQLEndpoint<FindByIdInput, FindOneResult<QuizAttempt>>;
        quizAttempts: GQLEndpoint<QuizAttemptSearchInput, FindManyResult<QuizAttempt>>;
    },
    Mutation: {
        quizAttemptCreate: GQLEndpoint<QuizAttemptCreateInput, CreateOneResult<QuizAttempt>>;
        quizAttemptUpdate: GQLEndpoint<QuizAttemptUpdateInput, UpdateOneResult<QuizAttempt>>;
    }
}

const objectType = "QuizAttempt";
export const QuizAttemptEndpoints: EndpointsQuizAttempt = {
    Query: {
        quizAttempt: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        quizAttempts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        quizAttemptCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        quizAttemptUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
