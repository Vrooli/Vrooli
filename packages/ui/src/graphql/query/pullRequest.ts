import { gql } from 'graphql-tag';
import { pullRequestFields } from 'graphql/fragment';

export const pullRequestQuery = gql`
    ${pullRequestFields}
    query pullRequest($input: FindByIdInput!) {
        pullRequest(input: $input) {
            ...pullRequestFields
        }
    }
`