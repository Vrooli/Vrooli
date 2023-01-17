import { toMutation } from 'graphql/utils';

export const deleteOneOrManyEndpoint = {
    deleteOne: toMutation('deleteOne', 'DeleteOneInput', [], `success`),
    deleteMany: toMutation('deleteMany', 'DeleteManyInput', [], `count`),
}