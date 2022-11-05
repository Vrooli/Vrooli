import { CopyInput, DeleteManyInput, DeleteOneInput, FindByIdInput, FindByIdOrHandleInput, FindByVersionInput, ForkInput, VisibilityType } from "../../schema/types";
import { PrismaType, ReqForUserAuth } from "../../types";
import { CountInputBase, GraphQLInfo, ModelLogic, PartialGraphQLInfo, SearchInputBase } from "../types";

export type CopyHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: CopyInput;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
}

export type CountHelperProps<GraphQLModel, CountInput extends CountInputBase> = {
    input: CountInput;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
    where?: { [x: string]: any };
    visibility?: VisibilityType;
}

export type CreateHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
}

export type DeleteManyHelperProps = {
    input: DeleteManyInput;
    model: ModelLogic<any, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
}

export type DeleteOneHelperProps = {
    input: DeleteOneInput;
    model: ModelLogic<any, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
}

export type ForkHelperProps<GraphQLModelType> = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: ForkInput,
    /**
     * If false, the UI won't bug you about creating a pull request
     */
    intendToPullRequest: boolean,
    model: ModelLogic<GraphQLModelType, any, any>,
    prisma: PrismaType,
    req: ReqForUserAuth;
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
    model: ModelLogic<GraphQLModel, SearchInput, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
}

export type ReadOneHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: FindByIdInput | FindByIdOrHandleInput | FindByVersionInput;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
}

export type UpdateHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    req: ReqForUserAuth;
    where?: (obj: any) => { [x: string]: any };
}