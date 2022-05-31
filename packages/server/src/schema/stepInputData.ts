/**
 * User log (e.g. created a project, completed a routine)
 */
import { gql } from 'apollo-server-express';
import { IWrap } from 'types';
import { Count, DeleteManyInput, FindByIdInput, StepInputData, StepInputDataCreateInput, StepInputDataSearchInput, StepInputDataSearchResult, StepInputDataSortBy, StepInputDataUpdateInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE } from '@local/shared';

export const typeDef = gql`
    enum StepInputDataSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }
 
    # TODO only stores data entered for auto-generated forms. Also need to store 
    # decision data somewhere
    input StepInputDataCreateInput {
        stepId: ID!
        runId: ID!
        nodeId: ID!
        routineId: ID!
        subroutineId: ID
        inputsCreate: [StepInputDataInputsCreateInput!]
    }

    input StepInputDataInputsCreateInput {
        inputId: ID!
        standardId: ID
        name: String!
        value: String!
    }

    input StepInputDataUpdateInput {
        stepId: ID!
        inputsCreate: [StepInputDataInputsCreateInput!]
        inputsUpdate: [StepInputDataInputsUpdateInput!]
    }

    input StepInputDataInputsUpdateInput {
        inputId: ID!
        value: String!
    }

    type StepInputData {
        id: ID!
        stepId: ID!
        runId: ID!
        nodeId: ID!
        routineId: ID!
        subroutineId: ID
        inputs: [StepInputDataInput!]!
    }

    type StepInputDataInput {
        id: ID!
        inputId: ID!
        standardId: ID
        name: String!
        value: String!
    }

    input StepInputDataSearchInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        ids: [ID!]
        nodeId: ID
        runId: ID
        sortBy: StepInputDataSortBy
        stepId: ID
        subroutineId: ID
        take: Int
    }

    type StepInputDataSearchResult {
        pageInfo: PageInfo!
        edges: [StepInputDataEdge!]!
    }

    type StepInputDataEdge {
        cursor: String!
        node: StepInputData!
    }

    extend type Query {
        stepInputData(input: FindByIdInput!): StepInputData
        stepInputDatas(input: StepInputDataSearchInput!): StepInputDataSearchResult!
    }

    extend type Mutation {
        stepInputDataCreate(input: StepInputDataCreateInput!): StepInputData!
        stepInputDataUpdate(input: StepInputDataUpdateInput!): StepInputData!
        stepInputDataDeleteMany(input: DeleteManyInput): Count!
    }
`

export const resolvers = {
    StepInputDataSortBy: StepInputDataSortBy,
    Query: {
        stepInputData: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<StepInputData> => {
            await rateLimit({ context, info, max: 5000, byAccount: true });
            throw new CustomError(CODE.NotImplemented);
        },
        stepInputDatas: async (_parent: undefined, { input }: IWrap<StepInputDataSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<StepInputDataSearchResult> => {
            await rateLimit({ context, info, max: 5000, byAccount: true });
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        stepInputDataCreate: async (_parent: undefined, { input }: IWrap<StepInputDataCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<StepInputData> => {
            await rateLimit({ context, info, max: 5000, byAccount: true });
            throw new CustomError(CODE.NotImplemented);
        },
        stepInputDataUpdate: async (_parent: undefined, { input }: IWrap<StepInputDataUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<StepInputData> => {
            await rateLimit({ context, info, max: 5000, byAccount: true });
            throw new CustomError(CODE.NotImplemented);
        },
        stepInputDataDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ context, info, max: 5000, byAccount: true });
            throw new CustomError(CODE.NotImplemented);
        },
    }
}