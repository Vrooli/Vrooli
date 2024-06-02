import { AwardCategory, AwardSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { AwardEndpoints, EndpointsAward } from "../logic/award";

export const typeDef = gql`
    enum AwardSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
        ProgressAsc
        ProgressDesc
    }

    enum AwardCategory {
        AccountAnniversary
        AccountNew
        ApiCreate
        #CodeCreate
        SmartContractCreate
        CommentCreate
        IssueCreate
        NoteCreate
        ObjectBookmark
        ObjectReact
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
        StandardCreate
        Streak
        #TeamCreate
        #TeamJoin
        OrganizationCreate
        OrganizationJoin
        UserInvite
    }  

    type Award {
        id: ID!
        created_at: Date!
        updated_at: Date!
        timeCurrentTierCompleted: Date
        category: AwardCategory!
        progress: Int!
        title: String # Only a title if you've reached a tier.
        description: String # Only a description if you've reached a tier.
    }

    input AwardSearchInput {
        after: String
        ids: [ID!]
        sortBy: AwardSortBy
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type AwardSearchResult {
        pageInfo: PageInfo!
        edges: [AwardEdge!]!
    }

    type AwardEdge {
        cursor: String!
        node: Award!
    }

    extend type Query {
        awards(input: AwardSearchInput!): AwardSearchResult!
    }
`;

export const resolvers: {
    AwardCategory: typeof AwardCategory;
    AwardSortBy: typeof AwardSortBy;
    Query: EndpointsAward["Query"];
} = {
    AwardCategory,
    AwardSortBy,
    ...AwardEndpoints,
};
