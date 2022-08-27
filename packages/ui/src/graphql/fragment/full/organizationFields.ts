import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationFields on Organization {
        id
        created_at
        handle
        isOpenToNewMembers
        isPrivate
        isStarred
        stars
        permissionsOrganization {
            canAddMembers
            canDelete
            canEdit
            canStar
            canReport
            isMember
        }
        resourceLists {
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
        roles {
            id
            created_at
            updated_at
            title
            translations {
                id
                language
                description
            }
        }
        tags {
            tag
            translations {
                id
                language
                description
            }
        }
        translations {
            id
            language
            bio
            name
        }
    }
`