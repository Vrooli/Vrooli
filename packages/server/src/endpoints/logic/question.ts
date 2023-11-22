import { FindByIdInput, Question, QuestionCreateInput, QuestionSearchInput, QuestionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsQuestion = {
    Query: {
        question: GQLEndpoint<FindByIdInput, FindOneResult<Question>>;
        questions: GQLEndpoint<QuestionSearchInput, FindManyResult<Question>>;
    },
    Mutation: {
        questionCreate: GQLEndpoint<QuestionCreateInput, CreateOneResult<Question>>;
        questionUpdate: GQLEndpoint<QuestionUpdateInput, UpdateOneResult<Question>>;
    }
}

const objectType = "Question";
export const QuestionEndpoints: EndpointsQuestion = {
    Query: {
        question: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        questions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        questionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        questionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
