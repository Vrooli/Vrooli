import { FindByIdInput, Quiz, QuizCreateInput, QuizSearchInput, QuizSearchResult, QuizUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsQuiz = {
    findOne: ApiEndpoint<FindByIdInput, Quiz>;
    findMany: ApiEndpoint<QuizSearchInput, QuizSearchResult>;
    createOne: ApiEndpoint<QuizCreateInput, Quiz>;
    updateOne: ApiEndpoint<QuizUpdateInput, Quiz>;
}

const objectType = "Quiz";
export const quiz: EndpointsQuiz = {
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
