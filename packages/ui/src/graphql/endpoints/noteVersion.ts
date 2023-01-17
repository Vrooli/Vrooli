import { noteVersionFields as fullFields, listNoteVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const noteVersionEndpoint = {
    findOne: toQuery('noteVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('noteVersions', 'NoteVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('noteVersionCreate', 'NoteVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('noteVersionUpdate', 'NoteVersionUpdateInput', [fullFields], `...fullFields`)
}