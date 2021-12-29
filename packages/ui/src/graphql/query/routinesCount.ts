import { gql } from 'graphql-tag';

export const routinesCountQuery = gql`
    query routinesCount {
        routinesCount
    }
`