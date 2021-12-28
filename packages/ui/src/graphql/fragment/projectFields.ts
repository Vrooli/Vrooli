import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment projectFields on Project {
        id
        name
        description
        created_at
    }
`