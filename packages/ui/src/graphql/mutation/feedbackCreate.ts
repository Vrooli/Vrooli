import { gql } from 'graphql-tag';

export const feedbackCreateMutation = gql`
    mutation feedbackCreate($input: FeedbackInput!) {
        feedbackCreate(input: $input) {
            success
        }
    }
`