import { gql } from 'graphql-tag';
import { tagFields } from '.';

export const organizationFields = gql`
    ${tagFields}
    fragment organizationFields on Organization {
        id
        name
        description
        created_at
        tags {
            ...tagFields
        }
    }
`