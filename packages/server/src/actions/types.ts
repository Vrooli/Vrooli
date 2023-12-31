import { CopyInput, DeleteManyInput, DeleteOneInput, GqlModelType, VisibilityType } from "@local/shared";
import { Request } from "express";
import { CountInputBase, GraphQLInfo, PartialGraphQLInfo } from "../builders/types";
import { PrismaType, SessionData, SessionUserToken } from "../types";

export type CountHelperProps<CountInput extends CountInputBase> = {
    input: CountInput;
    objectType: `${GqlModelType}`;
    prisma: PrismaType;
    req: { session: SessionData };
    where?: { [x: string]: any };
    visibility?: VisibilityType;
}

export type CreateOneHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: `${GqlModelType}`;
    prisma: PrismaType;
    req: { session: SessionData };
}

export type CreateManyHelperProps = Omit<CreateOneHelperProps, "input"> & {
    input: any[];
}

export type DeleteManyHelperProps = {
    input: DeleteManyInput;
    prisma: PrismaType;
    req: { session: SessionData };
}

export type DeleteOneHelperProps = {
    input: DeleteOneInput;
    prisma: PrismaType;
    req: { session: SessionData };
}

export type CopyHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: CopyInput,
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
    req: { session: SessionData },
}

export type ReadManyHelperProps<
    Input extends { [x: string]: any }
> = {
    additionalQueries?: { [x: string]: any };
    /**
     * Decides if queried data should be called. Defaults to true. 
     * You may want to set this to false if you are calling readManyHelper multiple times, so you can do this 
     * later in one call
     */
    addSupplemental?: boolean;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: Input;
    objectType: `${GqlModelType}`;
    prisma: PrismaType;
    req: { session: { languages: string[], users?: SessionUserToken[] } };
    visibility?: VisibilityType;
}

type FindUniqueInput = {
    id?: string | null | undefined;
    handle?: string | null | undefined;
    handleRoot?: string | null | undefined;
    idRoot?: string | null | undefined;
}
export type ReadOneHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: FindUniqueInput;
    objectType: `${GqlModelType}`;
    prisma: PrismaType;
    req: { session: { languages: string[], users?: SessionUserToken[] } };
}

export type RelBuilderHelperProps<
    IsAdd extends boolean,
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    RelName extends string,
    Input extends { [key in RelName]: any },
> = {
    data: Input
    isAdd: IsAdd,
    isOneToOne: IsOneToOne,
    isRequired: IsRequired,
    linkVersion?: boolean,
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
    relationshipName: RelName,
    userData: SessionUserToken,
}

export type UpdateOneHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: GqlModelType | `${GqlModelType}`;
    prisma: PrismaType;
    req: Request;
}

export type UpdateManyHelperProps = Omit<UpdateOneHelperProps, "input"> & {
    input: any[];
}
