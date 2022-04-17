import { gql } from 'graphql-tag';

export const findHandlesQuery = gql`
    query findHandles($input: FindHandlesInput!) {
        findHandles(input: $input)
    }
`