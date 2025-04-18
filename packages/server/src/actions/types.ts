import { CopyInput, DeleteManyInput, DeleteOneInput, ModelType, SessionUser, VisibilityType } from "@local/shared";
import { RequestService } from "../auth/request.js";
import { CountInputBase, PartialApiInfo } from "../builders/types.js";
import { EmbeddableType } from "../utils/embeddings/types.js";

export type CudAdditionalData = Record<string, any>;

export type CountHelperProps<CountInput extends CountInputBase> = {
    input: CountInput;
    objectType: `${ModelType}`;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
    where?: { [x: string]: any };
    visibility?: VisibilityType;
}

export type CreateOneHelperProps = {
    additionalData?: CudAdditionalData;
    info: PartialApiInfo;
    input: any;
    objectType: `${ModelType}`;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
}

export type CreateManyHelperProps = Omit<CreateOneHelperProps, "input"> & {
    input: any[];
}

export type DeleteManyHelperProps = {
    input: DeleteManyInput;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
}

export type DeleteOneHelperProps = {
    input: DeleteOneInput;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
}

export type CopyHelperProps = {
    info: PartialApiInfo,
    input: CopyInput,
    objectType: `${ModelType}`,
    req: Parameters<typeof RequestService.assertRequestFrom>[0],
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
    info: PartialApiInfo;
    input: Input;
    objectType: `${ModelType}`;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
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
    info: PartialApiInfo;
    input: FindUniqueInput;
    objectType: `${ModelType}`;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
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
    objectType: `${ModelType}`,
    relationshipName: RelName,
    userData: SessionUser,
}

export type UpdateOneHelperProps = {
    additionalData?: CudAdditionalData;
    info: PartialApiInfo;
    input: any;
    objectType: ModelType | `${ModelType}`;
    req: Parameters<typeof RequestService.assertRequestFrom>[0];
}

export type UpdateManyHelperProps = Omit<UpdateOneHelperProps, "input"> & {
    input: any[];
}
