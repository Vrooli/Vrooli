import { countPartial, successPartial } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const deleteOneOrManyEndpoint = {
    deleteOne: toMutation('deleteOne', 'DeleteOneInput', successPartial, 'full'),
    deleteMany: toMutation('deleteMany', 'DeleteManyInput', countPartial, 'full'),
}