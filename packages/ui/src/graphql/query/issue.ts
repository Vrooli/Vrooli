import { gql } from 'graphql-tag';
import { issueFields } from 'graphql/fragment';

export const issueQuery = gql`
    ${issueFields}
    query issue($input: FindByIdInput!) {
        issue(input: $input) {
            ...issueFields
        }
    }
`