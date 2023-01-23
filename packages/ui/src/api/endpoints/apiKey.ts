import { apiKeyPartial, successPartial } from '../partial';
import { toMutation } from '../utils';

export const apiKeyEndpoint = {
    create: toMutation('apiKeyCreate', 'ApiKeyCreateInput', apiKeyPartial, 'full'),
    update: toMutation('apiKeyUpdate', 'ApiKeyUpdateInput', apiKeyPartial, 'full'),
    deleteOne: toMutation('apiKeyDeleteOne', 'ApiKeyDeleteOneInput', successPartial, 'full'),
    validate: toMutation('apiKeyValidate', 'ApiKeyValidateInput', apiKeyPartial, 'full'),
}