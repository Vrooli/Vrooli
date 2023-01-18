import { noteVersionFields as fullFields, listNoteVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const noteVersionEndpoint = {
    findOne: toQuery('noteVersion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('noteVersions', 'NoteVersionSearchInput', toSearch(listFields)),
    create: toMutation('noteVersionCreate', 'NoteVersionCreateInput', fullFields[1]),
    update: toMutation('noteVersionUpdate', 'NoteVersionUpdateInput', fullFields[1])
}