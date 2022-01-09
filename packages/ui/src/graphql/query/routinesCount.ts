import { gql } from 'graphql-tag';

export const routinesCountQuery = gql`
    query routinesCount($input: RoutineCountInput!) {
        routinesCount(input: $input)
    }
`