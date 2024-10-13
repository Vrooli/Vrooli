import { CopyType } from "@local/shared";
import { CopyEndpoints, EndpointsCopy } from "../logic/copy";

export const typeDef = `#graphql
    enum CopyType {
        ApiVersion
        CodeVersion
        NoteVersion
        ProjectVersion
        RoutineVersion
        StandardVersion
        Team
    }  
 
    input CopyInput {
        id: ID!
        intendToPullRequest: Boolean!
        objectType: CopyType!
    }

    type CopyResult {
        apiVersion: ApiVersion
        codeVersion: CodeVersion
        noteVersion: NoteVersion
        projectVersion: ProjectVersion
        routineVersion: RoutineVersion
        standardVersion: StandardVersion
        team: Team
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
