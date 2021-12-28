import { gql } from 'graphql-tag';

export const feedbackAddMutation = gql`
    mutation feedbackAdd($input: FeedbackInput!) {
        feedbackAdd(input: $input) {
            success
        }
    }
`