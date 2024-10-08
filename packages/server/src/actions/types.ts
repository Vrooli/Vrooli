import { CopyInput, DeleteManyInput, DeleteOneInput, GqlModelType, VisibilityType } from "@local/shared";
import { Request } from "express";
import { CountInputBase, GraphQLInfo, PartialGraphQLInfo } from "../builders/types";
import { SessionData, SessionUserToken } from "../types";
import { EmbeddableType } from "../utils/embeddings/types";

export type CudAdditionalData = Record<string, any>;

export type CountHelperProps<CountInput extends CountInputBase> = {
    input: CountInput;
    objectType: `${GqlModelType}`;
    req: { session: SessionData };
    where?: { [x: string]: any };
    visibility?: VisibilityType;
}

export type CreateOneHelperProps = {
    additionalData?: CudAdditionalData;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: `${GqlModelType}`;
    req: { session: SessionData };
}

export type CreateManyHelperProps = Omit<CreateOneHelperProps, "input"> & {
    input: any[];
}

export type DeleteManyHelperProps = {
    input: DeleteManyInput;
    req: { session: SessionData };
}

export type DeleteOneHelperProps = {
    input: DeleteOneInput;
    req: { session: SessionData };
}

export type CopyHelperProps = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: CopyInput,
    objectType: `${GqlModelType}`,
    req: { session: SessionData },
}

export type ReadManyHelperProps<
    Input extends { [x: string]: any }
> = {
    additionalQueries?: { [x: string]: any };
    /**
     * Decides if supplemental data (e.g. if you've bookmarked an item) should be called. Defaults to true. 
     * You may want to set this to false if you are calling readManyHelper multiple times, so you can do this 
     * later in one call
     */
    addSupplemental?: boolean;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: Input;
    objectType: `${GqlModelType}`;
    req: { session: { languages: string[], users?: SessionUserToken[] } };
    visibility?: VisibilityType;
}

export type ReadManyWithEmbeddingsHelperProps<
    Input extends { [x: string]: any }
> = Pick<ReadManyHelperProps<Input>, "info" | "input" | "req" | "visibility"> & {
    /**
     * If you want just the IDs of the results, the results without supplemental fields 
     * (i.e. what would be returned from readManyHelper with "addSupplemental" set to false), 
     * or the full results with supplemental fields.
     */
    fetchMode?: "ids" | "noSupplemental" | "full";
    objectType: EmbeddableType;
};

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
    relationshipName: RelName,
    userData: SessionUserToken,
}

export type UpdateOneHelperProps = {
    additionalData?: CudAdditionalData;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    objectType: GqlModelType | `${GqlModelType}`;
    req: Request;
}

export type UpdateManyHelperProps = Omit<UpdateOneHelperProps, "input"> & {
    input: any[];
}
