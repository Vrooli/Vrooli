import gql from 'graphql-tag';
import { Tag_list } from '../fragments/Tag_list';

export const organizationFindOne = gql`${Tag_list}

query organization($input: FindByIdInput!) {
  organization(input: $input) {
    roles {
        members {
            id
            created_at
            updated_at
            isAdmin
            permissions
            organization {
                id
                handle
                you {
                    canAddMembers
                    canDelete
                    canStar
                    canReport
                    canUpdate
                    canRead
                    isStarred
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
            user {
                id
                name
                handle
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
    stars
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
        canStar
        canReport
        canUpdate
        canRead
        isStarred
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

export const organizationFindMany = gql`${Tag_list}

query organizations($input: OrganizationSearchInput!) {
  organizations(input: $input) {
    edges {
        cursor
        node {
            id
            handle
            created_at
            updated_at
            isOpenToNewMembers
            isPrivate
            commentsCount
            membersCount
            reportsCount
            stars
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
                canStar
                canReport
                canUpdate
                canRead
                isStarred
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

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
            organization {
                id
                handle
                you {
                    canAddMembers
                    canDelete
                    canStar
                    canReport
                    canUpdate
                    canRead
                    isStarred
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
            user {
                id
                name
                handle
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
    stars
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
        canStar
        canReport
        canUpdate
        canRead
        isStarred
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

export const organizationUpdate = gql`${Tag_list}

mutation organizationUpdate($input: OrganizationUpdateInput!) {
  organizationUpdate(input: $input) {
    roles {
        members {
            id
            created_at
            updated_at
            isAdmin
            permissions
            organization {
                id
                handle
                you {
                    canAddMembers
                    canDelete
                    canStar
                    canReport
                    canUpdate
                    canRead
                    isStarred
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
            user {
                id
                name
                handle
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
    stars
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
        canStar
        canReport
        canUpdate
        canRead
        isStarred
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

