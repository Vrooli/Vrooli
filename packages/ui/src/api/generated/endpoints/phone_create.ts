import gql from 'graphql-tag';

export const phoneCreate = gql`
mutation phoneCreate($input: PhoneCreateInput!) {
  phoneCreate(input: $input) {
    id
    phoneNumber
    verified
  }
}`;

