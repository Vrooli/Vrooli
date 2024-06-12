import { FindByIdInput, QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
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
        quizAttempt: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        quizAttempts: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        quizAttemptCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        quizAttemptUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
