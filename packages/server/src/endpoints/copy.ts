import { gql } from 'apollo-server-express';
import { CopyInput, CopyResult } from '@shared/consts';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { CopyType } from '@shared/consts';
import { copyHelper } from '../actions';
import { lowercaseFirstLetter } from '../builders';

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
        organization: Organization
        project: Project
        routine: Routine
        standard: Standard
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
            return { [lowercaseFirstLetter(input.objectType)]: result };
        }
    }
}