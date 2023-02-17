export const Bookmark_list = `fragment Bookmark_list on Bookmark {
id
to {
    ... on Api {
        ...Api_list
    }
    ... on Comment {
        ...Comment_list
    }
    ... on Issue {
        ...Issue_list
    }
    ... on Note {
        ...Note_list
    }
    ... on Organization {
        ...Organization_list
    }
    ... on Post {
        ...Post_list
    }
    ... on Project {
        ...Project_list
    }
    ... on Question {
        ...Question_list
    }
    ... on QuestionAnswer {
        ...QuestionAnswer_list
    }
    ... on Quiz {
        ...Quiz_list
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
    ... on Tag {
        ...Tag_list
    }
    ... on User {
        ...User_list
    }
}
}`;