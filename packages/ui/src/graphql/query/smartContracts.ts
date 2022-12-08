import { gql } from 'graphql-tag';
import { listSmartContractFields } from 'graphql/fragment';

export const smartContractsQuery = gql`
    ${listSmartContractFields}
    query smartContracts($input: SmartContractSearchInput!) {
        smartContracts(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listSmartContractFields
                }
            }
        }
    }
`