import { gql } from 'graphql-tag';

export const profileFields = gql`
    fragment profileResourceListFields on ResourceList {
        id
        created_at
        index
        usedFor
        translations {
            id
            language
            description
            title
        }
        resources {
            id
            created_at
            index
            link
            updated_at
            usedFor
            translations {
                id
                language
                description
                title
            }
        }
    }
    fragment profileFields on Profile {
        id
        handle
        name
        emails {
            id
            emailAddress
            receivesAccountUpdates
            receivesBusinessUpdates
            verified
        }
        wallets {
            id
            name
            publicAddress
            stakingAddress
            handles {
                id
                handle
            }
            verified
        }
        theme
        translations {
            id
            language
            bio
        }
        starredTags {
            id
            tag
            created_at
            stars
            isStarred
            translations {
                id
                language
                description
            }
        }
        hiddenTags {
            isBlur
            tag {
                id
                tag
                created_at
                stars
                isStarred
                translations {
                    id
                    language
                    description
                }
            }
        }
        resourceLists {
            ...profileResourceListFields
        }
    }
`