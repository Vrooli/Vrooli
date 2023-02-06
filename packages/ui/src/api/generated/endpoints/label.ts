import gql from 'graphql-tag';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const labelFindOne = gql`${Organization_nav}
${User_nav}

query label($input: FindByIdInput!) {
  label(input: $input) {
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
        canUpdate
    }
  }
}`;

export const labelFindMany = gql`${Organization_nav}
${User_nav}

query labels($input: LabelSearchInput!) {
  labels(input: $input) {
    edges {
        cursor
        node {
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
                canUpdate
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const labelCreate = gql`${Organization_nav}
${User_nav}

mutation labelCreate($input: LabelCreateInput!) {
  labelCreate(input: $input) {
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
        canUpdate
    }
  }
}`;

export const labelUpdate = gql`${Organization_nav}
${User_nav}

mutation labelUpdate($input: LabelUpdateInput!) {
  labelUpdate(input: $input) {
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
        canUpdate
    }
  }
}`;

