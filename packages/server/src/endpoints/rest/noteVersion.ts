import { noteVersion_create, noteVersion_findMany, noteVersion_findOne, noteVersion_update } from "../generated";
import { NoteVersionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const NoteVersionRest = setupRoutes({
    "/noteVersion/:id": {
        get: [NoteVersionEndpoints.Query.noteVersion, noteVersion_findOne],
        put: [NoteVersionEndpoints.Mutation.noteVersionUpdate, noteVersion_update],
    },
    "/noteVersions": {
        get: [NoteVersionEndpoints.Query.noteVersions, noteVersion_findMany],
    },
    "/noteVersion": {
        post: [NoteVersionEndpoints.Mutation.noteVersionCreate, noteVersion_create],
    },
});
