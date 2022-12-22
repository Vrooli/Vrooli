import { apiVersionFields as fullFields, listApiVersionFields as listFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const apiVersionEndpoint = {
    findOne: toGql('query', 'apiVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toGql('query', 'apiVersions', 'ApiVersionSearchInput', [listFields], `
        pageInfo {
            endCursor
            hasNextPage
        }
        edges {
            cursor
            node {
                ...listFields
            }
        }
    `),
    create: toGql('mutation', 'apiVersionCreate', 'ApiVersionCreateInput', [fullFields], `...fullFields`),
    update: toGql('mutation', 'apiVersionUpdate', 'ApiVersionUpdateInput', [fullFields], `...fullFields`)
}