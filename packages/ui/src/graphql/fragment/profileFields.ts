import { gql } from 'graphql-tag';

export const profileFields = gql`
    fragment profileFields on Profile {
        id
        username
        bio
        emails {
            id
            emailAddress
            receivesAccountUpdates
            receivesBusinessUpdates
        }
        wallets {
            publicAddress
            verified
        }
        theme
        starredTags {
            id
            tag
            description
            created_at
            stars
            isStarred
        }
        hiddenTags {
            isBlur
            tag {
                id
                tag
                description
                created_at
                stars
                isStarred
            }
        }
    }
`