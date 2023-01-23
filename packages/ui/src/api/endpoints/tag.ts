import { tagPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const tagEndpoint = {
    findOne: toQuery('tag', 'FindByIdInput', tagPartial, 'full'),
    findMany: toQuery('tags', 'TagSearchInput', ...toSearch(tagPartial)),
    create: toMutation('tagCreate', 'TagCreateInput', tagPartial, 'full'),
    update: toMutation('tagUpdate', 'TagUpdateInput', tagPartial, 'full')
}