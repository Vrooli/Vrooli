import { gql } from 'apollo-server-express';
import { AwardCategory } from '@shared/consts';

export const typeDef = gql`
    enum AwardCategory {
        AccountAnniversary
        AccountNew
        ApiCreate
        CommentCreate
        IssueCreate
        NoteCreate
        ObjectBookmark
        ObjectVote
        OrganizationCreate
        OrganizationJoin
        PostCreate
        ProjectCreate
        PullRequestCreate
        PullRequestComplete
        QuestionAnswer
        QuestionCreate
        QuizPass
        ReportEnd
        ReportContribute
        Reputation
        RunRoutine
        RunProject
        RoutineCreate
        SmartContractCreate
        StandardCreate
        Streak
        UserInvite
    }  

    type Award {
        id: ID!
        created_at: Date!
        updated_at: Date!
        timeCurrentTierCompleted: Date
        category: AwardCategory!
        progress: Int!
    }
`

export const resolvers: {
    AwardCategory: typeof AwardCategory;
} = {
    AwardCategory: AwardCategory
}