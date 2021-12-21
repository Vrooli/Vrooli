import { gql } from 'graphql-tag';

export const resourceFields = gql`
    fragment resourceFields on Resource {
        id
        name
        description
        link
        displayUrl
    }
`