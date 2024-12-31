import { endpointsNoteVersion } from "@local/shared";
import { noteVersion_create, noteVersion_findMany, noteVersion_findOne, noteVersion_update } from "../generated";
import { NoteVersionEndpoints } from "../logic/noteVersion";
import { setupRoutes } from "./base";

export const NoteVersionRest = setupRoutes([
    [endpointsNoteVersion.findOne, NoteVersionEndpoints.Query.noteVersion, noteVersion_findOne],
    [endpointsNoteVersion.findMany, NoteVersionEndpoints.Query.noteVersions, noteVersion_findMany],
    [endpointsNoteVersion.createOne, NoteVersionEndpoints.Mutation.noteVersionCreate, noteVersion_create],
    [endpointsNoteVersion.updateOne, NoteVersionEndpoints.Mutation.noteVersionUpdate, noteVersion_update],
]);
