import { apiKeyFields as fullFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const apiKeyEndpoint = {
    create: toMutation('apiKeyCreate', 'ApiKeyCreateInput', [fullFields], `...fullFields`),
    update: toMutation('apiKeyUpdate', 'ApiKeyUpdateInput', [fullFields], `...fullFields`),
    deleteOne: toMutation('apiKeyDeleteOne', 'ApiKeyDeleteOneInput', [], `success`),
    validate: toMutation('apiKeyValidate', 'ApiKeyValidateInput', [fullFields], `...fullFields`),
}