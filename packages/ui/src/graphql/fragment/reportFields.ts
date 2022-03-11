import { gql } from 'graphql-tag';

export const reportFields = gql`
    fragment reportFields on Report {
        id
        language
        reason
        details
        isOwn
    }
`