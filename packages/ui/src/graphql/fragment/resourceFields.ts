import { gql } from 'graphql-tag';

export const resourceFields = gql`
    fragment resourceFields on Resource {
        id
        title
        description
        link
        displayUrl
    }
`