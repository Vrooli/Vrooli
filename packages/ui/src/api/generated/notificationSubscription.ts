import gql from 'graphql-tag';

export const findOne = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
        id
        created_at
        updated_at
        callLink
        commentsCount
        documentationLink
        forksCount
        isLatest
        isPrivate
        reportsCount
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_nav on Organization {
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

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
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

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Comment_list on Comment {
    translations {
        id
        language
        text
    }
    id
    created_at
    updated_at
    commentedOn {
        ... on ApiVersion {
            ...Api_nav
        }
        ... on Issue {
            ...Issue_nav
        }
        ... on NoteVersion {
            ...NoteVersion_nav
        }
        ... on Post {
            ...Post_nav
        }
        ... on ProjectVersion {
            ...ProjectVersion_nav
        }
        ... on PullRequest {
            ...PullRequest_nav
        }
        ... on Question {
            ...Question_common
        }
        ... on QuestionAnswer {
            ...QuestionAnswer_common
        }
        ... on RoutineVersion {
            ...RoutineVersion_nav
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_nav
        }
        ... on StandardVersion {
            ...StandardVersion_nav
        }
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    score
    stars
    reportsCount
    you {
        canDelete
        canEdit
        canStar
        canReply
        canReport
        canVote
        isStarred
        isUpvoted
    }
}

fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
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
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
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

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
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

fragment SmartContractVersion_nav on SmartContractVersion {
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
    }
}

fragment StandardVersion_nav on StandardVersion {
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
    }
}

fragment Issue_list on Issue {
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
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Meeting_list on Meeting {
    labels {
        ...Label_list
    }
    translations {
        id
        language
        description
        link
        name
    }
    id
    openToAnyoneWithInvite
    showOnOrganizationProfile
    timeZone
    eventStart
    eventEnd
    recurring
    recurrStart
    recurrEnd
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
    restrictedToRoles {
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
        translations {
            id
            language
            description
        }
    }
    attendeesCount
    invitesCount
    you {
        canDelete
        canEdit
        canInvite
    }
}

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_list on Organization {
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

fragment Project_list on Project {
    versions {
        directories {
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
        runsCount
        simplicity
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment PullRequest_list on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
    id
    created_at
    updated_at
    callLink
    commentsCount
    documentationLink
    forksCount
    isLatest
    isPrivate
    reportsCount
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment ProjectVersion_list on ProjectVersion {
    directories {
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
    runsCount
    simplicity
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Routine_list on Routine {
    versions {
        translations {
            id
            language
            description
            instructions
            name
        }
        id
        created_at
        updated_at
        completedAt
        complexity
        isAutomatable
        isComplete
        isDeleted
        isLatest
        isPrivate
        simplicity
        timesStarted
        timesCompleted
        smartContractCallData
        apiCallData
        versionIndex
        versionLabel
        commentsCount
        directoryListingsCount
        forksCount
        inputsCount
        nodesCount
        nodeLinksCount
        outputsCount
        reportsCount
        you {
            runs {
                inputs {
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
                                canEdit
                                canReport
                                canUse
                                canView
                            }
                        }
                    }
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
                    subroutine {
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
                inputsCount
                wasRunAutomaticaly
                organization {
                    ...Organization_nav
                }
                runRoutineSchedule {
                    labels {
                        ...Label_full
                    }
                    id
                    attemptAutomatic
                    maxAutomaticAttempts
                    timeZone
                    windowStart
                    windowEnd
                    recurrStart
                    recurrEnd
                    translations {
                        id
                        language
                        description
                        name
                    }
                }
                user {
                    ...User_nav
                }
                you {
                    canDelete
                    canEdit
                    canView
                }
            }
            canComment
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canView
            canVote
        }
    }
    id
    created_at
    updated_at
    isInternal
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canComment
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
    translations {
        id
        language
        description
        instructions
        name
    }
    id
    created_at
    updated_at
    completedAt
    complexity
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    simplicity
    timesStarted
    timesCompleted
    smartContractCallData
    apiCallData
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    inputsCount
    nodesCount
    nodeLinksCount
    outputsCount
    reportsCount
    you {
        runs {
            inputs {
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
                            canEdit
                            canReport
                            canUse
                            canView
                        }
                    }
                }
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
                subroutine {
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
            inputsCount
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            runRoutineSchedule {
                labels {
                    ...Label_full
                }
                id
                attemptAutomatic
                maxAutomaticAttempts
                timeZone
                windowStart
                windowEnd
                recurrStart
                recurrEnd
                translations {
                    id
                    language
                    description
                    name
                }
            }
            user {
                ...User_nav
            }
            you {
                canDelete
                canEdit
                canView
            }
        }
        canComment
        canCopy
        canDelete
        canEdit
        canStar
        canReport
        canRun
        canView
        canVote
    }
}

fragment SmartContract_list on SmartContract {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment SmartContractVersion_list on SmartContractVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Standard_list on Standard {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment StandardVersion_list on StandardVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Question_list on Question {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Quiz_list on Quiz {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    isCompleted
    score
    stars
    views
    attemptsCount
    quizQuestionsCount
    project {
        id
        isPrivate
    }
    routine {
        id
        isInternal
        isPrivate
    }
    you {
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Report_list on Report {
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
}


query notificationSubscription($input: FindByIdInput!) {
  notificationSubscription(input: $input) {
    id
    created_at
    silent
    object {
        ... on Api {
            ...Api_list
        }
        ... on Comment {
            ...Comment_list
        }
        ... on Issue {
            ...Issue_list
        }
        ... on Meeting {
            ...Meeting_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Organization {
            ...Organization_list
        }
        ... on Project {
            ...Project_list
        }
        ... on PullRequest {
            ...PullRequest_list
        }
        ... on Question {
            ...Question_list
        }
        ... on Quiz {
            ...Quiz_list
        }
        ... on Report {
            ...Report_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
  }
}`;

export const findMany = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
        id
        created_at
        updated_at
        callLink
        commentsCount
        documentationLink
        forksCount
        isLatest
        isPrivate
        reportsCount
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_nav on Organization {
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

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
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

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Comment_list on Comment {
    translations {
        id
        language
        text
    }
    id
    created_at
    updated_at
    commentedOn {
        ... on ApiVersion {
            ...Api_nav
        }
        ... on Issue {
            ...Issue_nav
        }
        ... on NoteVersion {
            ...NoteVersion_nav
        }
        ... on Post {
            ...Post_nav
        }
        ... on ProjectVersion {
            ...ProjectVersion_nav
        }
        ... on PullRequest {
            ...PullRequest_nav
        }
        ... on Question {
            ...Question_common
        }
        ... on QuestionAnswer {
            ...QuestionAnswer_common
        }
        ... on RoutineVersion {
            ...RoutineVersion_nav
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_nav
        }
        ... on StandardVersion {
            ...StandardVersion_nav
        }
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    score
    stars
    reportsCount
    you {
        canDelete
        canEdit
        canStar
        canReply
        canReport
        canVote
        isStarred
        isUpvoted
    }
}

fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
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
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
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

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
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

fragment SmartContractVersion_nav on SmartContractVersion {
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
    }
}

fragment StandardVersion_nav on StandardVersion {
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
    }
}

fragment Issue_list on Issue {
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
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Meeting_list on Meeting {
    labels {
        ...Label_list
    }
    translations {
        id
        language
        description
        link
        name
    }
    id
    openToAnyoneWithInvite
    showOnOrganizationProfile
    timeZone
    eventStart
    eventEnd
    recurring
    recurrStart
    recurrEnd
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
    restrictedToRoles {
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
        translations {
            id
            language
            description
        }
    }
    attendeesCount
    invitesCount
    you {
        canDelete
        canEdit
        canInvite
    }
}

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_list on Organization {
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

fragment Project_list on Project {
    versions {
        directories {
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
        runsCount
        simplicity
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment PullRequest_list on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
    id
    created_at
    updated_at
    callLink
    commentsCount
    documentationLink
    forksCount
    isLatest
    isPrivate
    reportsCount
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment ProjectVersion_list on ProjectVersion {
    directories {
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
    runsCount
    simplicity
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Routine_list on Routine {
    versions {
        translations {
            id
            language
            description
            instructions
            name
        }
        id
        created_at
        updated_at
        completedAt
        complexity
        isAutomatable
        isComplete
        isDeleted
        isLatest
        isPrivate
        simplicity
        timesStarted
        timesCompleted
        smartContractCallData
        apiCallData
        versionIndex
        versionLabel
        commentsCount
        directoryListingsCount
        forksCount
        inputsCount
        nodesCount
        nodeLinksCount
        outputsCount
        reportsCount
        you {
            runs {
                inputs {
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
                                canEdit
                                canReport
                                canUse
                                canView
                            }
                        }
                    }
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
                    subroutine {
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
                inputsCount
                wasRunAutomaticaly
                organization {
                    ...Organization_nav
                }
                runRoutineSchedule {
                    labels {
                        ...Label_full
                    }
                    id
                    attemptAutomatic
                    maxAutomaticAttempts
                    timeZone
                    windowStart
                    windowEnd
                    recurrStart
                    recurrEnd
                    translations {
                        id
                        language
                        description
                        name
                    }
                }
                user {
                    ...User_nav
                }
                you {
                    canDelete
                    canEdit
                    canView
                }
            }
            canComment
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canView
            canVote
        }
    }
    id
    created_at
    updated_at
    isInternal
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canComment
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
    translations {
        id
        language
        description
        instructions
        name
    }
    id
    created_at
    updated_at
    completedAt
    complexity
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    simplicity
    timesStarted
    timesCompleted
    smartContractCallData
    apiCallData
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    inputsCount
    nodesCount
    nodeLinksCount
    outputsCount
    reportsCount
    you {
        runs {
            inputs {
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
                            canEdit
                            canReport
                            canUse
                            canView
                        }
                    }
                }
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
                subroutine {
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
            inputsCount
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            runRoutineSchedule {
                labels {
                    ...Label_full
                }
                id
                attemptAutomatic
                maxAutomaticAttempts
                timeZone
                windowStart
                windowEnd
                recurrStart
                recurrEnd
                translations {
                    id
                    language
                    description
                    name
                }
            }
            user {
                ...User_nav
            }
            you {
                canDelete
                canEdit
                canView
            }
        }
        canComment
        canCopy
        canDelete
        canEdit
        canStar
        canReport
        canRun
        canView
        canVote
    }
}

fragment SmartContract_list on SmartContract {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment SmartContractVersion_list on SmartContractVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Standard_list on Standard {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment StandardVersion_list on StandardVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Question_list on Question {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Quiz_list on Quiz {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    isCompleted
    score
    stars
    views
    attemptsCount
    quizQuestionsCount
    project {
        id
        isPrivate
    }
    routine {
        id
        isInternal
        isPrivate
    }
    you {
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Report_list on Report {
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
}


query notificationSubscriptions($input: NotificationSubscriptionSearchInput!) {
  notificationSubscriptions(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            silent
            object {
                ... on Api {
                    ...Api_list
                }
                ... on Comment {
                    ...Comment_list
                }
                ... on Issue {
                    ...Issue_list
                }
                ... on Meeting {
                    ...Meeting_list
                }
                ... on Note {
                    ...Note_list
                }
                ... on Organization {
                    ...Organization_list
                }
                ... on Project {
                    ...Project_list
                }
                ... on PullRequest {
                    ...PullRequest_list
                }
                ... on Question {
                    ...Question_list
                }
                ... on Quiz {
                    ...Quiz_list
                }
                ... on Report {
                    ...Report_list
                }
                ... on Routine {
                    ...Routine_list
                }
                ... on SmartContract {
                    ...SmartContract_list
                }
                ... on Standard {
                    ...Standard_list
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

export const create = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
        id
        created_at
        updated_at
        callLink
        commentsCount
        documentationLink
        forksCount
        isLatest
        isPrivate
        reportsCount
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_nav on Organization {
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

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
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

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Comment_list on Comment {
    translations {
        id
        language
        text
    }
    id
    created_at
    updated_at
    commentedOn {
        ... on ApiVersion {
            ...Api_nav
        }
        ... on Issue {
            ...Issue_nav
        }
        ... on NoteVersion {
            ...NoteVersion_nav
        }
        ... on Post {
            ...Post_nav
        }
        ... on ProjectVersion {
            ...ProjectVersion_nav
        }
        ... on PullRequest {
            ...PullRequest_nav
        }
        ... on Question {
            ...Question_common
        }
        ... on QuestionAnswer {
            ...QuestionAnswer_common
        }
        ... on RoutineVersion {
            ...RoutineVersion_nav
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_nav
        }
        ... on StandardVersion {
            ...StandardVersion_nav
        }
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    score
    stars
    reportsCount
    you {
        canDelete
        canEdit
        canStar
        canReply
        canReport
        canVote
        isStarred
        isUpvoted
    }
}

fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
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
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
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

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
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

fragment SmartContractVersion_nav on SmartContractVersion {
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
    }
}

fragment StandardVersion_nav on StandardVersion {
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
    }
}

fragment Issue_list on Issue {
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
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Meeting_list on Meeting {
    labels {
        ...Label_list
    }
    translations {
        id
        language
        description
        link
        name
    }
    id
    openToAnyoneWithInvite
    showOnOrganizationProfile
    timeZone
    eventStart
    eventEnd
    recurring
    recurrStart
    recurrEnd
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
    restrictedToRoles {
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
        translations {
            id
            language
            description
        }
    }
    attendeesCount
    invitesCount
    you {
        canDelete
        canEdit
        canInvite
    }
}

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_list on Organization {
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

fragment Project_list on Project {
    versions {
        directories {
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
        runsCount
        simplicity
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment PullRequest_list on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
    id
    created_at
    updated_at
    callLink
    commentsCount
    documentationLink
    forksCount
    isLatest
    isPrivate
    reportsCount
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment ProjectVersion_list on ProjectVersion {
    directories {
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
    runsCount
    simplicity
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Routine_list on Routine {
    versions {
        translations {
            id
            language
            description
            instructions
            name
        }
        id
        created_at
        updated_at
        completedAt
        complexity
        isAutomatable
        isComplete
        isDeleted
        isLatest
        isPrivate
        simplicity
        timesStarted
        timesCompleted
        smartContractCallData
        apiCallData
        versionIndex
        versionLabel
        commentsCount
        directoryListingsCount
        forksCount
        inputsCount
        nodesCount
        nodeLinksCount
        outputsCount
        reportsCount
        you {
            runs {
                inputs {
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
                                canEdit
                                canReport
                                canUse
                                canView
                            }
                        }
                    }
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
                    subroutine {
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
                inputsCount
                wasRunAutomaticaly
                organization {
                    ...Organization_nav
                }
                runRoutineSchedule {
                    labels {
                        ...Label_full
                    }
                    id
                    attemptAutomatic
                    maxAutomaticAttempts
                    timeZone
                    windowStart
                    windowEnd
                    recurrStart
                    recurrEnd
                    translations {
                        id
                        language
                        description
                        name
                    }
                }
                user {
                    ...User_nav
                }
                you {
                    canDelete
                    canEdit
                    canView
                }
            }
            canComment
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canView
            canVote
        }
    }
    id
    created_at
    updated_at
    isInternal
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canComment
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
    translations {
        id
        language
        description
        instructions
        name
    }
    id
    created_at
    updated_at
    completedAt
    complexity
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    simplicity
    timesStarted
    timesCompleted
    smartContractCallData
    apiCallData
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    inputsCount
    nodesCount
    nodeLinksCount
    outputsCount
    reportsCount
    you {
        runs {
            inputs {
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
                            canEdit
                            canReport
                            canUse
                            canView
                        }
                    }
                }
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
                subroutine {
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
            inputsCount
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            runRoutineSchedule {
                labels {
                    ...Label_full
                }
                id
                attemptAutomatic
                maxAutomaticAttempts
                timeZone
                windowStart
                windowEnd
                recurrStart
                recurrEnd
                translations {
                    id
                    language
                    description
                    name
                }
            }
            user {
                ...User_nav
            }
            you {
                canDelete
                canEdit
                canView
            }
        }
        canComment
        canCopy
        canDelete
        canEdit
        canStar
        canReport
        canRun
        canView
        canVote
    }
}

fragment SmartContract_list on SmartContract {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment SmartContractVersion_list on SmartContractVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Standard_list on Standard {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment StandardVersion_list on StandardVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Question_list on Question {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Quiz_list on Quiz {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    isCompleted
    score
    stars
    views
    attemptsCount
    quizQuestionsCount
    project {
        id
        isPrivate
    }
    routine {
        id
        isInternal
        isPrivate
    }
    you {
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Report_list on Report {
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
}


mutation notificationSubscriptionCreate($input: NotificationSubscriptionCreateInput!) {
  notificationSubscriptionCreate(input: $input) {
    id
    created_at
    silent
    object {
        ... on Api {
            ...Api_list
        }
        ... on Comment {
            ...Comment_list
        }
        ... on Issue {
            ...Issue_list
        }
        ... on Meeting {
            ...Meeting_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Organization {
            ...Organization_list
        }
        ... on Project {
            ...Project_list
        }
        ... on PullRequest {
            ...PullRequest_list
        }
        ... on Question {
            ...Question_list
        }
        ... on Quiz {
            ...Quiz_list
        }
        ... on Report {
            ...Report_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
  }
}`;

export const update = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
        id
        created_at
        updated_at
        callLink
        commentsCount
        documentationLink
        forksCount
        isLatest
        isPrivate
        reportsCount
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_nav on Organization {
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

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
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

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Comment_list on Comment {
    translations {
        id
        language
        text
    }
    id
    created_at
    updated_at
    commentedOn {
        ... on ApiVersion {
            ...Api_nav
        }
        ... on Issue {
            ...Issue_nav
        }
        ... on NoteVersion {
            ...NoteVersion_nav
        }
        ... on Post {
            ...Post_nav
        }
        ... on ProjectVersion {
            ...ProjectVersion_nav
        }
        ... on PullRequest {
            ...PullRequest_nav
        }
        ... on Question {
            ...Question_common
        }
        ... on QuestionAnswer {
            ...QuestionAnswer_common
        }
        ... on RoutineVersion {
            ...RoutineVersion_nav
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_nav
        }
        ... on StandardVersion {
            ...StandardVersion_nav
        }
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    score
    stars
    reportsCount
    you {
        canDelete
        canEdit
        canStar
        canReply
        canReport
        canVote
        isStarred
        isUpvoted
    }
}

fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
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
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
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

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
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

fragment SmartContractVersion_nav on SmartContractVersion {
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
    }
}

fragment StandardVersion_nav on StandardVersion {
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
    }
}

fragment Issue_list on Issue {
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
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Meeting_list on Meeting {
    labels {
        ...Label_list
    }
    translations {
        id
        language
        description
        link
        name
    }
    id
    openToAnyoneWithInvite
    showOnOrganizationProfile
    timeZone
    eventStart
    eventEnd
    recurring
    recurrStart
    recurrEnd
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
    restrictedToRoles {
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
        translations {
            id
            language
            description
        }
    }
    attendeesCount
    invitesCount
    you {
        canDelete
        canEdit
        canInvite
    }
}

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Organization_list on Organization {
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

fragment Project_list on Project {
    versions {
        directories {
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
        runsCount
        simplicity
        versionIndex
        versionLabel
        you {
            canComment
            canCopy
            canDelete
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment PullRequest_list on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
    id
    created_at
    updated_at
    callLink
    commentsCount
    documentationLink
    forksCount
    isLatest
    isPrivate
    reportsCount
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment ProjectVersion_list on ProjectVersion {
    directories {
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
    runsCount
    simplicity
    versionIndex
    versionLabel
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Routine_list on Routine {
    versions {
        translations {
            id
            language
            description
            instructions
            name
        }
        id
        created_at
        updated_at
        completedAt
        complexity
        isAutomatable
        isComplete
        isDeleted
        isLatest
        isPrivate
        simplicity
        timesStarted
        timesCompleted
        smartContractCallData
        apiCallData
        versionIndex
        versionLabel
        commentsCount
        directoryListingsCount
        forksCount
        inputsCount
        nodesCount
        nodeLinksCount
        outputsCount
        reportsCount
        you {
            runs {
                inputs {
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
                                canEdit
                                canReport
                                canUse
                                canView
                            }
                        }
                    }
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
                    subroutine {
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
                inputsCount
                wasRunAutomaticaly
                organization {
                    ...Organization_nav
                }
                runRoutineSchedule {
                    labels {
                        ...Label_full
                    }
                    id
                    attemptAutomatic
                    maxAutomaticAttempts
                    timeZone
                    windowStart
                    windowEnd
                    recurrStart
                    recurrEnd
                    translations {
                        id
                        language
                        description
                        name
                    }
                }
                user {
                    ...User_nav
                }
                you {
                    canDelete
                    canEdit
                    canView
                }
            }
            canComment
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canView
            canVote
        }
    }
    id
    created_at
    updated_at
    isInternal
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canComment
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
    translations {
        id
        language
        description
        instructions
        name
    }
    id
    created_at
    updated_at
    completedAt
    complexity
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    simplicity
    timesStarted
    timesCompleted
    smartContractCallData
    apiCallData
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    inputsCount
    nodesCount
    nodeLinksCount
    outputsCount
    reportsCount
    you {
        runs {
            inputs {
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
                            canEdit
                            canReport
                            canUse
                            canView
                        }
                    }
                }
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
                subroutine {
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
            inputsCount
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            runRoutineSchedule {
                labels {
                    ...Label_full
                }
                id
                attemptAutomatic
                maxAutomaticAttempts
                timeZone
                windowStart
                windowEnd
                recurrStart
                recurrEnd
                translations {
                    id
                    language
                    description
                    name
                }
            }
            user {
                ...User_nav
            }
            you {
                canDelete
                canEdit
                canView
            }
        }
        canComment
        canCopy
        canDelete
        canEdit
        canStar
        canReport
        canRun
        canView
        canVote
    }
}

fragment SmartContract_list on SmartContract {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment SmartContractVersion_list on SmartContractVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Standard_list on Standard {
    versions {
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
            canEdit
            canReport
            canUse
            canView
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
    stars
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canEdit
        canStar
        canTransfer
        canView
        canVote
        isStarred
        isUpvoted
        isViewed
    }
}

fragment StandardVersion_list on StandardVersion {
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
        canEdit
        canReport
        canUse
        canView
    }
}

fragment Question_list on Question {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
}

fragment Quiz_list on Quiz {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    isCompleted
    score
    stars
    views
    attemptsCount
    quizQuestionsCount
    project {
        id
        isPrivate
    }
    routine {
        id
        isInternal
        isPrivate
    }
    you {
        canDelete
        canEdit
        canStar
        canView
        canVote
        isStarred
        isUpvoted
    }
}

fragment Report_list on Report {
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
}


mutation notificationSubscriptionUpdate($input: NotificationSubscriptionUpdateInput!) {
  notificationSubscriptionUpdate(input: $input) {
    id
    created_at
    silent
    object {
        ... on Api {
            ...Api_list
        }
        ... on Comment {
            ...Comment_list
        }
        ... on Issue {
            ...Issue_list
        }
        ... on Meeting {
            ...Meeting_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Organization {
            ...Organization_list
        }
        ... on Project {
            ...Project_list
        }
        ... on PullRequest {
            ...PullRequest_list
        }
        ... on Question {
            ...Question_list
        }
        ... on Quiz {
            ...Quiz_list
        }
        ... on Report {
            ...Report_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
  }
}`;

