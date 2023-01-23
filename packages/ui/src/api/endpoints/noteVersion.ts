import { noteVersionPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const noteVersionEndpoint = {
    findOne: toQuery('noteVersion', 'FindByIdInput', noteVersionPartial, 'full'),
    findMany: toQuery('noteVersions', 'NoteVersionSearchInput', ...toSearch(noteVersionPartial)),
    create: toMutation('noteVersionCreate', 'NoteVersionCreateInput', noteVersionPartial, 'full'),
    update: toMutation('noteVersionUpdate', 'NoteVersionUpdateInput', noteVersionPartial, 'full')
}