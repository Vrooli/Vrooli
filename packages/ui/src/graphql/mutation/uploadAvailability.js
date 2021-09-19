import { gql } from 'graphql-tag';

export const uploadAvailabilityMutation = gql`
    mutation uploadAvailability(
        $file: Upload!
    ) {
    uploadAvailability(
        file: $file
    )
}
`