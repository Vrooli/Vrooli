import gql from "graphql-tag";

export const phoneUpdate = gql`
mutation sendVerificationText($input: SendVerificationTextInput!) {
  sendVerificationText(input: $input) {
    success
  }
}`;

