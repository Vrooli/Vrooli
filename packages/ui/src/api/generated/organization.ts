import gql from 'graphql-tag';

export const findOne = gql`fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}


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
                    canEdit
                    canStar
                    canReport
                    canView
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
        canEdit
        canStar
        canReport
        canView
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

export const findMany = gql`fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}


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
                canEdit
                canStar
                canReport
                canView
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

export const create = gql`fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}


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
                    canEdit
                    canStar
                    canReport
                    canView
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
        canEdit
        canStar
        canReport
        canView
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

export const update = gql`fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}


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
                    canEdit
                    canStar
                    canReport
                    canView
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
        canEdit
        canStar
        canReport
        canView
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

