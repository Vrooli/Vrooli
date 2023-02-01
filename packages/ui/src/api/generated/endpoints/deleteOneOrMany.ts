import gql from 'graphql-tag';

export const deleteOneOrManyDeleteOne = gql`
mutation deleteOne($input: DeleteOneInput!) {
  deleteOne(input: $input) {
    success
  }
}`;

export const deleteOneOrManyDeleteMany = gql`
mutation deleteMany($input: DeleteManyInput!) {
  deleteMany(input: $input) {
    count
  }
}`;

