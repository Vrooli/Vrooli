import gql from "graphql-tag";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const commentFindMany = gql`${Organization_nav}
${User_nav}

query comments($input: CommentSearchInput!) {
  comments(input: $input) {
    endCursor
    threads {
        childThreads {
            childThreads {
                comment {
                    translations {
                        id
                        language
                        text
                    }
                    id
                    created_at
                    updated_at
                    owner {
                        ... on Organization {
                            ...Organization_nav
                        }
                        ... on User {
                            ...User_nav
                        }
                    }
                    score
                    bookmarks
                    reportsCount
                    you {
                        canDelete
                        canBookmark
                        canReply
                        canReport
                        canUpdate
                        canReact
                        isBookmarked
                        reaction
                    }
                }
                endCursor
                totalInThread
            }
            comment {
                translations {
                    id
                    language
                    text
                }
                id
                created_at
                updated_at
                owner {
                    ... on Organization {
                        ...Organization_nav
                    }
                    ... on User {
                        ...User_nav
                    }
                }
                score
                bookmarks
                reportsCount
                you {
                    canDelete
                    canBookmark
                    canReply
                    canReport
                    canUpdate
                    canReact
                    isBookmarked
                    reaction
                }
            }
            endCursor
            totalInThread
        }
        comment {
            translations {
                id
                language
                text
            }
            id
            created_at
            updated_at
            owner {
                ... on Organization {
                    ...Organization_nav
                }
                ... on User {
                    ...User_nav
                }
            }
            score
            bookmarks
            reportsCount
            you {
                canDelete
                canBookmark
                canReply
                canReport
                canUpdate
                canReact
                isBookmarked
                reaction
            }
        }
        endCursor
        totalInThread
    }
    totalThreads
  }
}`;

