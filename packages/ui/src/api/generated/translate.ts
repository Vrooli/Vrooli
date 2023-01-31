import gql from 'graphql-tag';

export const translate = gql`
query translate($input: FindByIdInput!) {
  translate(input: $input) {
    fields
    language
  }
}`;

