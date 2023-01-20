import { tagPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const tagEndpoint = {
    findOne: toQuery('tag', 'FindByIdInput', tagPartial, 'full'),
    findMany: toQuery('tags', 'TagSearchInput', ...toSearch(tagPartial)),
    create: toMutation('tagCreate', 'TagCreateInput', tagPartial, 'full'),
    update: toMutation('tagUpdate', 'TagUpdateInput', tagPartial, 'full')
}