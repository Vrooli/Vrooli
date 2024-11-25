import { FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
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
        noteVersion: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        noteVersions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        noteVersionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        noteVersionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
