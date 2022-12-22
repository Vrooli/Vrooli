import { gql } from 'graphql-tag';
import { apiFields as fullFields, listApiFields as listFields } from 'graphql/fragment';

export const apiEndpoint = {
    findOne: gql`
        ${fullFields}
        query api($input: FindByIdInput!) {
            api(input: $input) {
                ...fullFields
            }
        }
    `,
    findMany: gql`
        ${listFields}
        query apis($input: ApiSearchInput!) {
            apis(input: $input) {
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
            }
        }
    `,
    create: gql`
        ${fullFields}
        mutation apiCreate($input: ApiCreateInput!) {
            apiCreate(input: $input) {
                ...fullFields
            }
        }
    `,
    update: gql`
        ${fullFields}
        mutation apiUpdate($input: ApiUpdateInput!) {
            apiUpdate(input: $input) {
                ...fullFields
            }
        }
    `
}