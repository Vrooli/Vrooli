import { gql } from 'graphql-tag';
import { tagFields } from '.';

export const standardFields = gql`
    ${tagFields}
    fragment standardFields on Standard {
        id
        name
        description
        type
        schema
        default
        isFile
        created_at
        tags {
            ...tagFields
        }
    }
`