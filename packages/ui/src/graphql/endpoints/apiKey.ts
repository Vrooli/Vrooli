import { apiKeyFields as fullFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const apiKeyEndpoint = {
    create: toGql('mutation', 'apiKeyCreate', 'ApiKeyCreateInput', [fullFields], `...fullFields`),
    update: toGql('mutation', 'apiKeyUpdate', 'ApiKeyUpdateInput', [fullFields], `...fullFields`),
    deleteOne: toGql('mutation', 'apiKeyDeleteOne', 'ApiKeyDeleteOneInput', [], `success`),
    validate: toGql('mutation', 'apiKeyValidate', 'ApiKeyValidateInput', [fullFields], `...fullFields`),
}