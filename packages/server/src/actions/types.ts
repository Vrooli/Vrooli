import { Request } from "express";
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, FindByIdOrHandleInput, FindByVersionInput, ForkInput, SessionUser, VisibilityType } from "../schema/types";
import { PrismaType, RecursivePartial } from "../types";
import { AniedModelLogic, GraphQLModelType } from "../models/types";
import { CountInputBase, GraphQLInfo, PartialGraphQLInfo } from "../builders/types";

export type CopyHelperProps<GraphQLModel extends { [x: string]: any}> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: ForkInput;
    model: AniedModelLogic<GraphQLModel>;
    prisma: PrismaType;
    req: Request;
}

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

export interface CUDResult<GraphQLObject extends { [x: string]: any}> {
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
    input: DeleteOneInput;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
}

export type ForkHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: ForkInput,
    objectType: GraphQLModelType,
    prisma: PrismaType,
    req: Request;
}

export type ReadManyHelperProps<GQLObject extends { [x: string]: any}> = {
    additionalQueries?: { [x: string]: any };
    /**
     * Decides if queried data should be called. Defaults to true. 
     * You may want to set this to false if you are calling readManyHelper multiple times, so you can do this 
     * later in one call
     */
    addSupplemental?: boolean;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: AniedModelLogic<GQLObject>;
    prisma: PrismaType;
    req: { languages: string[], users?: SessionUser[] };
}

export type ReadOneHelperProps<GQLObject extends { [x: string]: any}> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: FindByIdInput | FindByIdOrHandleInput | FindByVersionInput;
    model: AniedModelLogic<GQLObject>;
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