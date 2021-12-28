import { gql } from 'graphql-tag';

export const commentFields = gql`
    fragment commentFields on Comment {
        id
        text
        created_at
        updated_at
        userId
        organizationId
        projectId
        resourceId
        routineId
        standardId
    }
`