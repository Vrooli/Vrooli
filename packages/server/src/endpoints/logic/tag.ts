import { FindByIdInput, Tag, TagCreateInput, TagSearchInput, TagUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyWithEmbeddingsHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsTag = {
    Query: {
        tag: GQLEndpoint<FindByIdInput, FindOneResult<Tag>>;
        tags: GQLEndpoint<TagSearchInput, FindManyResult<Tag>>;
    },
    Mutation: {
        tagCreate: GQLEndpoint<TagCreateInput, CreateOneResult<Tag>>;
        tagUpdate: GQLEndpoint<TagUpdateInput, UpdateOneResult<Tag>>;
    }
}

const objectType = "Tag";
export const TagEndpoints: EndpointsTag = {
    Query: {
        tag: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        tags: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyWithEmbeddingsHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        tagCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        tagUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
