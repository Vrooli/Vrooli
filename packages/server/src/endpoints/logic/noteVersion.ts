import { FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsNoteVersion = {
    Query: {
        noteVersion: GQLEndpoint<FindVersionInput, FindOneResult<NoteVersion>>;
        noteVersions: GQLEndpoint<NoteVersionSearchInput, FindManyResult<NoteVersion>>;
    },
    Mutation: {
        noteVersionCreate: GQLEndpoint<NoteVersionCreateInput, CreateOneResult<NoteVersion>>;
        noteVersionUpdate: GQLEndpoint<NoteVersionUpdateInput, UpdateOneResult<NoteVersion>>;
    }
}

const objectType = "NoteVersion";
export const NoteVersionEndpoints: EndpointsNoteVersion = {
    Query: {
        noteVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        noteVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        noteVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        noteVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
