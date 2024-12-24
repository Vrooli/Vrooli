import { FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsNoteVersion = {
    Query: {
        noteVersion: ApiEndpoint<FindVersionInput, FindOneResult<NoteVersion>>;
        noteVersions: ApiEndpoint<NoteVersionSearchInput, FindManyResult<NoteVersion>>;
    },
    Mutation: {
        noteVersionCreate: ApiEndpoint<NoteVersionCreateInput, CreateOneResult<NoteVersion>>;
        noteVersionUpdate: ApiEndpoint<NoteVersionUpdateInput, UpdateOneResult<NoteVersion>>;
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
