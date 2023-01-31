import gql from 'graphql-tag';

export const create = gql`
mutation phoneCreate($input: PhoneCreateInput!) {
  phoneCreate(input: $input) {
    id
    phoneNumber
    verified
  }
}`;

export const update = gql`
mutation sendVerificationText($input: SendVerificationTextInput!) {
  sendVerificationText(input: $input) {
    success
  }
}`;

