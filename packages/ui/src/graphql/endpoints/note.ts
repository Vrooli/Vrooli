import { noteFields as fullFields, listNoteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const noteEndpoint = {
    findOne: toQuery('note', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('notes', 'NoteSearchInput', toSearch(listFields)),
    create: toMutation('noteCreate', 'NoteCreateInput', fullFields[1]),
    update: toMutation('noteUpdate', 'NoteUpdateInput', fullFields[1])
}