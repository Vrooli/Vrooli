import { Request } from "express";
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, FindByIdOrHandleInput, FindByVersionInput, CopyInput, SessionUser, VisibilityType } from "../endpoints/types";
import { PrismaType, RecursivePartial } from "../types";
import { GraphQLModelType } from "../models/types";
import { CountInputBase, GraphQLInfo, PartialGraphQLInfo } from "../builders/types";

export type CountHelperProps<CountInput extends CountInputBase> = {
    input: CountInput;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
    where?: { [x: string]: any };
    visibility?: VisibilityType;
}

export type CreateHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
}

export interface CUDHelperInput {
    createMany?: { [x: string]: any }[] | null | undefined;
    deleteMany?: string[] | null | undefined,
    objectType: GraphQLModelType,
    partialInfo: PartialGraphQLInfo,
    prisma: PrismaType,
    updateMany?: { [x: string]: any }[] | null | undefined,
    userData: SessionUser,
}

export interface CUDResult<GraphQLObject extends { [x: string]: any }> {
    created?: RecursivePartial<GraphQLObject>[],
    updated?: RecursivePartial<GraphQLObject>[],
    deleted?: Count,
}

export type DeleteManyHelperProps = {
    input: DeleteManyInput;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
}

export type DeleteOneHelperProps = {
    input: Pick<DeleteOneInput, 'id'>;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
}

export type CopyHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: CopyInput,
    objectType: GraphQLModelType,
    prisma: PrismaType,
    req: Request;
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
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: { languages: string[], users?: SessionUser[] };
}

export type ReadOneHelperProps<GQLObject extends { [x: string]: any }> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: FindByIdInput | FindByIdOrHandleInput | FindByVersionInput;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: { languages: string[], users?: SessionUser[] };
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
    objectType: GraphQLModelType,
    prisma: PrismaType,
    relationshipName: RelName,
    userData: SessionUser,
}

export type UpdateHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
    where?: (obj: any) => { [x: string]: any };
}