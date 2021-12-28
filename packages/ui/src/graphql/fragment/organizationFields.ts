import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationFields on Organization {
        id
        name
        description
        created_at
    }
`