import { gql } from 'graphql-tag';
import { smartContractFields } from 'graphql/fragment';

export const smartContractQuery = gql`
    ${smartContractFields}
    query smartContract($input: FindByIdInput!) {
        smartContract(input: $input) {
            ...smartContractFields
        }
    }
`