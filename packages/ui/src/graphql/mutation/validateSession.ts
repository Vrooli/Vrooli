import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const validateSessionMutation = gql`
    ${sessionFields}
    mutation validateSession {
        validateSession {
            ...sessionFields
        }
    }
`