import gql from 'graphql-tag';
import { Tag_list } from '../fragments/Tag_list';

export const organizationCreate = gql`${Tag_list}

mutation organizationCreate($input: OrganizationCreateInput!) {
  organizationCreate(input: $input) {
    roles {
        members {
            id
            created_at
            updated_at
            isAdmin
            permissions
            roles {
                id
                created_at
                updated_at
                name
                permissions
                membersCount
                organization {
                    id
                    handle
                    you {
                        canAddMembers
                        canDelete
                        canBookmark
                        canReport
                        canUpdate
                        canRead
                        isBookmarked
                        isViewed
                        yourMembership {
                            id
                            created_at
                            updated_at
                            isAdmin
                            permissions
                        }
                    }
                }
                translations {
                    id
                    language
                    description
                }
            }
        }
        id
        created_at
        updated_at
        name
        permissions
        membersCount
        translations {
            id
            language
            description
        }
    }
    id
    handle
    created_at
    updated_at
    isOpenToNewMembers
    isPrivate
    commentsCount
    membersCount
    reportsCount
    bookmarks
    tags {
        ...Tag_list
    }
    translations {
        id
        language
        bio
        name
    }
    you {
        canAddMembers
        canDelete
        canBookmark
        canReport
        canUpdate
        canRead
        isBookmarked
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
  }
}`;

