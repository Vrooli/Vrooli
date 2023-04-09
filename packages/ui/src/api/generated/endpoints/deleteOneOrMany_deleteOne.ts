import gql from 'graphql-tag';

export const deleteOneOrManyDeleteOne = gql`
mutation deleteOne($input: DeleteOneInput!) {
  deleteOne(input: $input) {
    success
  }
}`;

