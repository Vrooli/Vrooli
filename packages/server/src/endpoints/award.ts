import { Award, AwardCategory, AwardSearchInput, AwardSortBy } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { readManyHelper } from '../actions';
import { rateLimit } from '../middleware';
import { FindManyResult, GQLEndpoint } from '../types';

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
        CommentCreate
        IssueCreate
        NoteCreate
        ObjectBookmark
        ObjectReact
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
`

const objectType = 'Award';
export const resolvers: {
    AwardCategory: typeof AwardCategory;
    AwardSortBy: typeof AwardSortBy;
    Query: {
        awards: GQLEndpoint<AwardSearchInput, FindManyResult<Award>>;
    },
} = {
    AwardCategory,
    AwardSortBy,
    Query: {
        awards: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}