import { CopyType } from "@local/shared";
import { gql } from "apollo-server-express";
import { CopyEndpoints, EndpointsCopy } from "../logic";

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
 `;

export const resolvers: {
    CopyType: typeof CopyType;
    Mutation: EndpointsCopy["Mutation"];
} = {
    CopyType,
    ...CopyEndpoints,
};
