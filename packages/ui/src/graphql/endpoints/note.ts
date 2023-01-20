import { notePartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const noteEndpoint = {
    findOne: toQuery('note', 'FindByIdInput', notePartial, 'full'),
    findMany: toQuery('notes', 'NoteSearchInput', ...toSearch(notePartial)),
    create: toMutation('noteCreate', 'NoteCreateInput', notePartial, 'full'),
    update: toMutation('noteUpdate', 'NoteUpdateInput', notePartial, 'full')
}