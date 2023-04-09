export const View_list = `fragment View_list on View {
id
to {
    ... on Api {
        ...Api_list
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
    ... on Routine {
        ...Routine_list
    }
    ... on SmartContract {
        ...SmartContract_list
    }
    ... on Standard {
        ...Standard_list
    }
    ... on User {
        ...User_list
    }
}
}`;