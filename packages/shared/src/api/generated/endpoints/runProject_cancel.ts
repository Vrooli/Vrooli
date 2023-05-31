import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Label_list } from "../fragments/Label_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const runProjectCancel = gql`${Label_full}
${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation runProjectCancel($input: RunProjectCancelInput!) {
  runProjectCancel(input: $input) {
    projectVersion {
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
        }
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
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                complexity
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
    }
    id
    isPrivate
    completedComplexity
    contextSwitches
    startedAt
    timeElapsed
    completedAt
    name
    status
    stepsCount
    organization {
        ...Organization_nav
    }
    schedule {
        labels {
            ...Label_full
        }
        id
        created_at
        updated_at
        startTime
        endTime
        timezone
        exceptions {
            id
            originalStartTime
            newStartTime
            newEndTime
        }
        recurrences {
            id
            recurrenceType
            interval
            dayOfWeek
            dayOfMonth
            month
            endDate
        }
    }
    user {
        ...User_nav
    }
    you {
        canDelete
        canUpdate
        canRead
    }
  }
}`;

