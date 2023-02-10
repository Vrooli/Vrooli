export const Question_list = `fragment Question_list on Question {
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
}`;