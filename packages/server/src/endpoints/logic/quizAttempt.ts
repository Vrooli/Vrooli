import { FindByIdInput, QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsQuizAttempt = {
    Query: {
        quizAttempt: ApiEndpoint<FindByIdInput, FindOneResult<QuizAttempt>>;
        quizAttempts: ApiEndpoint<QuizAttemptSearchInput, FindManyResult<QuizAttempt>>;
    },
    Mutation: {
        quizAttemptCreate: ApiEndpoint<QuizAttemptCreateInput, CreateOneResult<QuizAttempt>>;
        quizAttemptUpdate: ApiEndpoint<QuizAttemptUpdateInput, UpdateOneResult<QuizAttempt>>;
    }
}

const objectType = "QuizAttempt";
export const QuizAttemptEndpoints: EndpointsQuizAttempt = {
    Query: {
        quizAttempt: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        quizAttempts: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        quizAttemptCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        quizAttemptUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
