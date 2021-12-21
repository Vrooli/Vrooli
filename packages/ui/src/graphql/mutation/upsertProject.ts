import { gql } from 'graphql-tag';

export const upsertProjectMutation = gql`
    mutation upsertProject($input: ProjectInput!) {
        upsertProject(input: $input) {
            id
        }
    }
`