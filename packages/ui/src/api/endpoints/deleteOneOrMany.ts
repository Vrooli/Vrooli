import { countPartial, successPartial } from '../partial';
import { toMutation } from '../utils';

export const deleteOneOrManyEndpoint = {
    deleteOne: toMutation('deleteOne', 'DeleteOneInput', successPartial, 'full'),
    deleteMany: toMutation('deleteMany', 'DeleteManyInput', countPartial, 'full'),
}