import gql from 'graphql-tag';

export const standardVersionFindMany = gql`
query standardVersions($input: StandardVersionSearchInput!) {
  standardVersions(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                description
                jsonVariable
                name
            }
            id
            created_at
            updated_at
            isComplete
            isFile
            isLatest
            isPrivate
            default
            standardType
            props
            yup
            versionIndex
            versionLabel
            commentsCount
            directoryListingsCount
            forksCount
            reportsCount
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

