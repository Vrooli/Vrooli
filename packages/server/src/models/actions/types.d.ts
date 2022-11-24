import { Request } from "express";
import { CopyInput, DeleteManyInput, DeleteOneInput, FindByIdInput, FindByIdOrHandleInput, FindByVersionInput, ForkInput, SessionUser, VisibilityType } from "../../schema/types";
import { PrismaType } from "../../types";
import { CountInputBase, GraphQLInfo, GraphQLModelType, ModelLogic, PartialGraphQLInfo, SearchInputBase } from "../types";

export type CopyHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: CopyInput;
    model: ModelLogic<GraphQLModel, any, any, any>;
    prisma: PrismaType;
    req: Request;
}

export type CountHelperProps<GraphQLModel, CountInput extends CountInputBase> = {
    input: CountInput;
    model: ModelLogic<GraphQLModel, any, any, any>;
    prisma: PrismaType;
    req: Request;
    where?: { [x: string]: any };
    visibility?: VisibilityType;
}

export type CreateHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
}

export type DeleteManyHelperProps = {
    input: DeleteManyInput;
    model: ModelLogic<any, any, any, any>;
    prisma: PrismaType;
    req: Request;
}

export type DeleteOneHelperProps = {
    input: DeleteOneInput;
    model: ModelLogic<any, any, any, any>;
    prisma: PrismaType;
    req: Request;
}

export type ForkHelperProps<GraphQLModelType> = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: ForkInput,
    model: ModelLogic<GraphQLModelType, any, any, any>,
    prisma: PrismaType,
    req: Request;
}

export type ReadManyHelperProps<GraphQLModel, SearchInput extends SearchInputBase<any>> = {
    additionalQueries?: { [x: string]: any };
    /**
     * Decides if queried data should be called. Defaults to true. 
     * You may want to set this to false if you are calling readManyHelper multiple times, so you can do this 
     * later in one call
     */
    addSupplemental?: boolean;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, SearchInput, any, any>;
    prisma: PrismaType;
    req: { languages: string[], users?: SessionUser[] };
}

export type ReadOneHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: FindByIdInput | FindByIdOrHandleInput | FindByVersionInput;
    model: ModelLogic<GraphQLModel, any, any, any>;
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

export type UpdateHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: GraphQLModelType;
    prisma: PrismaType;
    req: Request;
    where?: (obj: any) => { [x: string]: any };
}