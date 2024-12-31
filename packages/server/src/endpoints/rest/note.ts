import { endpointsNote } from "@local/shared";
import { note_create, note_findMany, note_findOne, note_update } from "../generated";
import { NoteEndpoints } from "../logic/note";
import { setupRoutes } from "./base";

export const NoteRest = setupRoutes([
    [endpointsNote.findOne, NoteEndpoints.Query.note, note_findOne],
    [endpointsNote.findMany, NoteEndpoints.Query.notes, note_findMany],
    [endpointsNote.createOne, NoteEndpoints.Mutation.noteCreate, note_create],
    [endpointsNote.updateOne, NoteEndpoints.Mutation.noteUpdate, note_update],
]);
