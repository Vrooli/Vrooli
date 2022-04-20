import { gql } from 'graphql-tag';

export const logFields = gql`
    fragment logFields on Log {
        id
        timestamp
        action
        object1Type
        object1Id
        object2Type
        object2Id
        data
    }
`