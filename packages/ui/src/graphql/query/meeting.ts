import { gql } from 'graphql-tag';
import { meetingFields } from 'graphql/fragment';

export const meetingQuery = gql`
    ${meetingFields}
    query meeting($input: FindByIdInput!) {
        meeting(input: $input) {
            ...meetingFields
        }
    }
`