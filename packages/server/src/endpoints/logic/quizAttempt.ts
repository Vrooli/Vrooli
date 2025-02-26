import { FindByIdInput, QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptSearchResult, QuizAttemptUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsQuizAttempt = {
    findOne: ApiEndpoint<FindByIdInput, QuizAttempt>;
    findMany: ApiEndpoint<QuizAttemptSearchInput, QuizAttemptSearchResult>;
    createOne: ApiEndpoint<QuizAttemptCreateInput, QuizAttempt>;
    updateOne: ApiEndpoint<QuizAttemptUpdateInput, QuizAttempt>;
}

const objectType = "QuizAttempt";
export const quizAttempt: EndpointsQuizAttempt = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
