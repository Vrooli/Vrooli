export const Issue_list = `fragment Issue_list on Issue {
translations {
    id
    language
    description
    name
}
id
created_at
updated_at
closedAt
referencedVersionId
status
to {
    ... on Api {
        ...Api_nav
    }
    ... on Note {
        ...Note_nav
    }
    ... on Organization {
        ...Organization_nav
    }
    ... on Project {
        ...Project_nav
    }
    ... on Routine {
        ...Routine_nav
    }
    ... on SmartContract {
        ...SmartContract_nav
    }
    ... on Standard {
        ...Standard_nav
    }
}
commentsCount
reportsCount
score
bookmarks
views
labels {
    ...Label_common
}
you {
    canComment
    canDelete
    canBookmark
    canReport
    canUpdate
    canRead
    canVote
    isBookmarked
    isUpvoted
}
}`;