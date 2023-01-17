import { noteFields as fullFields, listNoteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const noteEndpoint = {
    findOne: toQuery('note', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('notes', 'NoteSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('noteCreate', 'NoteCreateInput', [fullFields], `...fullFields`),
    update: toMutation('noteUpdate', 'NoteUpdateInput', [fullFields], `...fullFields`)
}