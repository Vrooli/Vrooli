import { apiKeyFields as fullFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const apiKeyEndpoint = {
    create: toMutation('apiKeyCreate', 'ApiKeyCreateInput', fullFields[1]),
    update: toMutation('apiKeyUpdate', 'ApiKeyUpdateInput', fullFields[1]),
    deleteOne: toMutation('apiKeyDeleteOne', 'ApiKeyDeleteOneInput', `{ success }`),
    validate: toMutation('apiKeyValidate', 'ApiKeyValidateInput', fullFields[1]),
}