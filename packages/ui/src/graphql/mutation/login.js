import { gql } from 'graphql-tag';
import { customerSessionFields, orderFields, orderItemFields } from 'graphql/fragment';

export const loginMutation = gql`
    ${customerSessionFields}
    ${orderFields}
    ${orderItemFields}
    mutation login(
        $email: String
        $password: String
    ) {
    login(
        email: $email
        password: $password
    ) {
        ...customerSessionFields
        cart {
            ...orderFields
            items {
                ...orderItemFields
            }
        }
    }
}
`