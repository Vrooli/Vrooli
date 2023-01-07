import { gql } from 'apollo-server-express';
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Member, MemberSearchInput, MemberUpdateInput, MemberSortBy } from '@shared/consts';
import { rateLimit } from '../middleware';
import { readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum MemberSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input MemberUpdateInput {
        id: ID!
        isAdmin: Boolean
        permissions: String
    }
    type Member {
        type: GqlModelType!
        id: ID!
        created_at: Date!
        updated_at: Date!
        isAdmin: Boolean!
        permissions: String!
        organization: Organization!
        user: User!
    }

    input MemberSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isAdmin: Boolean
        roles: [String!]
        organizationId: ID
        userId: ID
        searchString: String
        sortBy: MemberSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MemberSearchResult {
        pageInfo: PageInfo!
        edges: [MemberEdge!]!
    }

    type MemberEdge {
        cursor: String!
        node: Member!
    }

    extend type Query {
        member(input: FindByIdInput!): Member
        members(input: MemberSearchInput!): MemberSearchResult!
    }

    extend type Mutation {
        memberUpdate(input: MemberUpdateInput!): Member!
    }
`

const objectType = 'Member';
export const resolvers: {
    MemberSortBy: typeof MemberSortBy;
    Query: {
        member: GQLEndpoint<FindByIdInput, FindOneResult<Member>>;
        members: GQLEndpoint<MemberSearchInput, FindManyResult<Member>>;
    },
    Mutation: {
        memberUpdate: GQLEndpoint<MemberUpdateInput, UpdateOneResult<Member>>;
    }
} = {
    MemberSortBy,
    Query: {
        member: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        members: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        memberUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}