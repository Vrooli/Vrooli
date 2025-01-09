import { FindByIdInput, QuizQuestionResponse, QuizQuestionResponseSearchInput, QuizQuestionResponseSearchResult } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsQuizQuestionResponse = {
    findOne: ApiEndpoint<FindByIdInput, QuizQuestionResponse>;
    findMany: ApiEndpoint<QuizQuestionResponseSearchInput, QuizQuestionResponseSearchResult>;
}

const objectType = "QuizQuestionResponse";
export const quizQuestionResponse: EndpointsQuizQuestionResponse = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
