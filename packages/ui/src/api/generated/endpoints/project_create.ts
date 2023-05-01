import gql from "graphql-tag";
import { Label_list } from "../fragments/Label_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const projectCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation projectCreate($input: ProjectCreateInput!) {
  projectCreate(input: $input) {
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
        }
    }
    versions {
        directories {
            children {
                id
                created_at
                updated_at
                childOrder
                isRoot
                projectVersion {
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
                    }
                }
            }
            childApiVersions {
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
                    details
                    name
                    summary
                }
            }
            childNoteVersions {
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
            childOrganizations {
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
            childProjectVersions {
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
                }
            }
            childRoutineVersions {
                id
                isAutomatable
                isComplete
                isDeleted
                isLatest
                isPrivate
                root {
                    id
                    isInternal
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    instructions
                    name
                }
                versionIndex
                versionLabel
            }
            childSmartContractVersions {
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
                    jsonVariable
                    name
                }
            }
            childStandardVersions {
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
                    jsonVariable
                    name
                }
            }
            parentDirectory {
                id
                created_at
                updated_at
                childOrder
                isRoot
                projectVersion {
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
                    }
                }
            }
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
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
                }
            }
        }
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
        translations {
            id
            language
            description
            name
        }
        versionNotes
        id
        created_at
        updated_at
        directoriesCount
        isLatest
        isPrivate
        reportsCount
        runProjectsCount
        simplicity
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
    stats {
        id
        periodStart
        periodEnd
        periodType
        directories
        apis
        notes
        organizations
        projects
        routines
        smartContracts
        standards
        runsStarted
        runsCompleted
        runCompletionTimeAverage
        runContextSwitchesAverage
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
        canReact
        isBookmarked
        isViewed
        reaction
    }
  }
}`;

