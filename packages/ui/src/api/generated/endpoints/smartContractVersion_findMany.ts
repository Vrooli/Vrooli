import gql from 'graphql-tag';

export const smartContractVersionFindMany = gql`
query smartContractVersions($input: SmartContractVersionSearchInput!) {
  smartContractVersions(input: $input) {
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
            isDeleted
            isLatest
            isPrivate
            default
            contractType
            content
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

