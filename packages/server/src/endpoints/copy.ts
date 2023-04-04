import { CopyInput, CopyResult, CopyType } from '@shared/consts';
import { lowercaseFirstLetter } from '@shared/utils';
import { gql } from 'apollo-server-express';
import { copyHelper } from '../actions';
import { rateLimit } from '../middleware';
import { GQLEndpoint } from '../types';

export const typeDef = gql`
    enum CopyType {
        ApiVersion
        NoteVersion
        Organization
        ProjectVersion
        RoutineVersion
        SmartContractVersion
        StandardVersion
    }  
 
    input CopyInput {
        id: ID!
        intendToPullRequest: Boolean!
        objectType: CopyType!
    }

    type CopyResult {
        apiVersion: ApiVersion
        noteVersion: NoteVersion
        organization: Organization
        projectVersion: ProjectVersion
        routineVersion: RoutineVersion
        smartContractVersion: SmartContractVersion
        standardVersion: StandardVersion
    }
 
    extend type Mutation {
        copy(input: CopyInput!): CopyResult!
    }
 `

export const resolvers: {
    CopyType: typeof CopyType;
    Mutation: {
        copy: GQLEndpoint<CopyInput, CopyResult>;
    }
} = {
    CopyType,
    Mutation: {
        copy: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            const result = await copyHelper({ info, input, objectType: input.objectType, prisma, req })
            return { __typename: 'CopyResult' as const, [lowercaseFirstLetter(input.objectType)]: result };
        }
    }
}