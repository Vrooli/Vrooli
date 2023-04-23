import { CopyType } from "@local/consts";
import { lowercaseFirstLetter } from "@local/utils";
import { gql } from "apollo-server-express";
import { copyHelper } from "../actions";
import { rateLimit } from "../middleware";
export const typeDef = gql `
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
 `;
export const resolvers = {
    CopyType,
    Mutation: {
        copy: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            const result = await copyHelper({ info, input, objectType: input.objectType, prisma, req });
            return { __typename: "CopyResult", [lowercaseFirstLetter(input.objectType)]: result };
        },
    },
};
//# sourceMappingURL=copy.js.map