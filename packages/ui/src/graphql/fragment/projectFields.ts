import { gql } from 'graphql-tag';
import { tagFields } from '.';

export const projectFields = gql`
    ${tagFields}
    fragment projectFields on Project {
        id
        name
        description
        created_at
        tags {
            ...tagFields
        }
    }
`