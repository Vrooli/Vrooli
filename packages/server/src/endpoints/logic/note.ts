import { FindByIdInput, Note, NoteCreateInput, NoteSearchInput, NoteUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsNote = {
    Query: {
        note: GQLEndpoint<FindByIdInput, FindOneResult<Note>>;
        notes: GQLEndpoint<NoteSearchInput, FindManyResult<Note>>;
    },
    Mutation: {
        noteCreate: GQLEndpoint<NoteCreateInput, CreateOneResult<Note>>;
        noteUpdate: GQLEndpoint<NoteUpdateInput, UpdateOneResult<Note>>;
    }
}

const objectType = "Note";
export const NoteEndpoints: EndpointsNote = {
    Query: {
        note: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        notes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        noteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        noteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
