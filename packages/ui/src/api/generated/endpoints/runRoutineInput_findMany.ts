import gql from 'graphql-tag';

export const runRoutineInputFindMany = gql`
query runRoutineInputs($input: RunRoutineInputSearchInput!) {
  runRoutineInputs(input: $input) {
    edges {
        cursor
        node {
            id
            data
            input {
                id
                index
                isRequired
                name
                standardVersion {
                    translations {
                        id
                        language
                        description
                        jsonVariable
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

