export const NoteVersion_list = `fragment NoteVersion_list on NoteVersion {
root {
    id
    created_at
    updated_at
    isPrivate
    issuesCount
    labels {
        ...Label_list
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    permissions
    questionsCount
    score
    bookmarks
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canBookmark
        canTransfer
        canUpdate
        canRead
        canReact
        isBookmarked
        isViewed
        reaction
    }
}
translations {
    id
    language
    description
    name
    text
}
id
created_at
updated_at
isLatest
isPrivate
reportsCount
versionIndex
versionLabel
you {
    canComment
    canCopy
    canDelete
    canReport
    canUpdate
    canUse
    canRead
}
}`;
