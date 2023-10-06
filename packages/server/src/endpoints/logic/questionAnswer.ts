import { FindByIdInput, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsQuestionAnswer = {
    Query: {
        questionAnswer: GQLEndpoint<FindByIdInput, FindOneResult<QuestionAnswer>>;
        questionAnswers: GQLEndpoint<QuestionAnswerSearchInput, FindManyResult<QuestionAnswer>>;
    },
    Mutation: {
        questionAnswerCreate: GQLEndpoint<QuestionAnswerCreateInput, CreateOneResult<QuestionAnswer>>;
        questionAnswerUpdate: GQLEndpoint<QuestionAnswerUpdateInput, UpdateOneResult<QuestionAnswer>>;
        questionAnswerMarkAsAccepted: GQLEndpoint<FindByIdInput, UpdateOneResult<QuestionAnswer>>;
    }
}

const objectType = "QuestionAnswer";
export const QuestionAnswerEndpoints: EndpointsQuestionAnswer = {
    Query: {
        questionAnswer: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        questionAnswers: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        questionAnswerCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        questionAnswerUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        questionAnswerMarkAsAccepted: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            throw new CustomError("000", "NotImplemented", ["en"]);
        },
    },
};
