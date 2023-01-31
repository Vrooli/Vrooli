import gql from 'graphql-tag';

export const emailCreate = gql`
mutation emailCreate($input: PhoneCreateInput!) {
  emailCreate(input: $input) {
    id
    emailAddress
    verified
  }
}`;

export const emailVerify = gql`
mutation sendVerificationEmail($input: SendVerificationEmailInput!) {
  sendVerificationEmail(input: $input) {
    success
  }
}`;

