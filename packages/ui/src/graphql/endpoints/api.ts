import { apiFields as fullFields, listApiFields as listFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const apiEndpoint = {
    findOne: toGql('query', 'api', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toGql('query', 'apis', 'ApiSearchInput', [listFields], `
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
    create: toGql('mutation', 'apiCreate', 'ApiCreateInput', [fullFields], `...fullFields`),
    update: toGql('mutation', 'apiUpdate', 'ApiUpdateInput', [fullFields], `...fullFields`)
}