import gql from "graphql-tag";

export const translateTranslate = gql`
query translate($input: FindByIdInput!) {
  translate(input: $input) {
    fields
    language
  }
}`;

