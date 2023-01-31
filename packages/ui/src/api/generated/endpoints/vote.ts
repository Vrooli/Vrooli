import gql from 'graphql-tag';

export const voteFindMany = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${Comment_list}
...${Api_nav}
...${Issue_nav}
...${NoteVersion_nav}
...${Post_nav}
...${ProjectVersion_nav}
...${PullRequest_nav}
...${Question_common}
...${Note_nav}
...${Project_nav}
...${Routine_nav}
...${SmartContract_nav}
...${Standard_nav}
...${QuestionAnswer_common}
...${RoutineVersion_nav}
...${SmartContractVersion_nav}
...${StandardVersion_nav}
...${Issue_list}
...${Label_common}
...${Note_list}
...${Post_list}
...${Project_list}
...${Question_list}
...${QuestionAnswer_list}
...${Quiz_list}
...${Routine_list}
...${Label_full}
...${SmartContract_list}
...${Standard_list}

query votes($input: VoteSearchInput!) {
  votes(input: $input) {
    edges {
        cursor
        node {
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
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const voteVote = gql`
mutation vote($input: VoteInput!) {
  vote(input: $input) {
    success
  }
}`;

