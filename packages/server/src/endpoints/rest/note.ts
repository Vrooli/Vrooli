import { note_create, note_findMany, note_findOne, note_update } from "../generated";
import { NoteEndpoints } from "../logic/note";
import { setupRoutes } from "./base";

export const NoteRest = setupRoutes({
    "/note/:id": {
        get: [NoteEndpoints.Query.note, note_findOne],
        put: [NoteEndpoints.Mutation.noteUpdate, note_update],
    },
    "/notes": {
        get: [NoteEndpoints.Query.notes, note_findMany],
    },
    "/note": {
        post: [NoteEndpoints.Mutation.noteCreate, note_create],
    },
});
