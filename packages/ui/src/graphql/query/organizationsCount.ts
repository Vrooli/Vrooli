import { gql } from 'graphql-tag';

export const organizationsCountQuery = gql`
    query organizationsCount($input: OrganizationCountInput!) {
        organizationsCount(input: $input)
    }
`