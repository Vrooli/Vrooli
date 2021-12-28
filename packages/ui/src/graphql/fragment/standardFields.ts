import { gql } from 'graphql-tag';

export const standardFields = gql`
    fragment standardFields on Standard {
        id
        name
        description
        type
        schema
        default
        isFile
        created_at
    }
`