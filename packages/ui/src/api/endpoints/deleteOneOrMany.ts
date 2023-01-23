import { countPartial, successPartial } from 'api/partial';
import { toMutation } from 'api/utils';

export const deleteOneOrManyEndpoint = {
    deleteOne: toMutation('deleteOne', 'DeleteOneInput', successPartial, 'full'),
    deleteMany: toMutation('deleteMany', 'DeleteManyInput', countPartial, 'full'),
}