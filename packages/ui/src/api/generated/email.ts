import gql from 'graphql-tag';

export const create = gql`
mutation emailCreate($input: PhoneCreateInput!) {
  emailCreate(input: $input) {
    id
    emailAddress
    verified
  }
}`;

export const verify = gql`
mutation sendVerificationEmail($input: SendVerificationEmailInput!) {
  sendVerificationEmail(input: $input) {
    success
  }
}`;

