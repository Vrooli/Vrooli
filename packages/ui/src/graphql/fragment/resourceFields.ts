import { gql } from 'graphql-tag';

export const resourceFields = gql`
    fragment resourceFields on Resource {
        id
        link
        usedFor
        translations {
            id
            language
            description
            title
        }
    }
`