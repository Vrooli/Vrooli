import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const noteVersionFindOne = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query noteVersion($input: FindVersionInput!) {
  noteVersion(input: $input) {
    pullRequest {
        translations {
            id
            language
            text
        }
        id
        created_at
        updated_at
        mergedOrRejectedAt
        commentsCount
        status
        createdBy {
            id
            name
            handle
        }
        you {
            canComment
            canDelete
            canReport
            canUpdate
        }
    }
    root {
        parent {
            id
            isLatest
            isPrivate
            versionIndex
            versionLabel
            root {
                id
                isPrivate
            }
            translations {
                id
                language
                description
                name
                text
            }
        }
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
            canVote
            isBookmarked
            isUpvoted
            isViewed
        }
    }
    translations {
        id
        language
        description
        name
        text
    }
    versionNotes
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
  }
}`;

export const noteVersionFindMany = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query noteVersions($input: NoteVersionSearchInput!) {
  noteVersions(input: $input) {
    edges {
        cursor
        node {
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
                    canVote
                    isBookmarked
                    isUpvoted
                    isViewed
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const noteVersionCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation noteVersionCreate($input: NoteVersionCreateInput!) {
  noteVersionCreate(input: $input) {
    pullRequest {
        translations {
            id
            language
            text
        }
        id
        created_at
        updated_at
        mergedOrRejectedAt
        commentsCount
        status
        createdBy {
            id
            name
            handle
        }
        you {
            canComment
            canDelete
            canReport
            canUpdate
        }
    }
    root {
        parent {
            id
            isLatest
            isPrivate
            versionIndex
            versionLabel
            root {
                id
                isPrivate
            }
            translations {
                id
                language
                description
                name
                text
            }
        }
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
            canVote
            isBookmarked
            isUpvoted
            isViewed
        }
    }
    translations {
        id
        language
        description
        name
        text
    }
    versionNotes
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
  }
}`;

export const noteVersionUpdate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation noteVersionUpdate($input: NoteVersionUpdateInput!) {
  noteVersionUpdate(input: $input) {
    pullRequest {
        translations {
            id
            language
            text
        }
        id
        created_at
        updated_at
        mergedOrRejectedAt
        commentsCount
        status
        createdBy {
            id
            name
            handle
        }
        you {
            canComment
            canDelete
            canReport
            canUpdate
        }
    }
    root {
        parent {
            id
            isLatest
            isPrivate
            versionIndex
            versionLabel
            root {
                id
                isPrivate
            }
            translations {
                id
                language
                description
                name
                text
            }
        }
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
            canVote
            isBookmarked
            isUpvoted
            isViewed
        }
    }
    translations {
        id
        language
        description
        name
        text
    }
    versionNotes
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
  }
}`;

