import { gql } from 'graphql-tag';

export const profileFields = gql`
    fragment profileFields on User {
        id
        handle
        isPrivate
        isPrivateApis
        isPrivateApisCreated
        isPrivateMemberships
        isPrivateOrganizationsCreated
        isPrivateProjects
        isPrivateProjectsCreated
        isPrivatePullRequests
        isPrivateQuestionsAnswered
        isPrivateQuestionsAsked
        isPrivateQuizzesCreated
        isPrivateRoles
        isPrivateRoutines
        isPrivateRoutinesCreated
        isPrivateRoutinesCreated
        isPrivateStandards
        isPrivateStandardsCreated
        isPrivateStars
        isPrivateVotes
        name
        emails {
            id
            emailAddress
            receivesAccountUpdates
            receivesBusinessUpdates
            verified
        }
        pushDevices {
            id
            expires
            name
        }
        wallets {
            id
            name
            publicAddress
            stakingAddress
            handles {
                id
                handle
            }
            verified
        }
        theme
        translations {
            id
            language
            bio
        }
        schedules {
            id
        }
    }
`