import { GqlModelType } from "@local/shared";
import { PrismaCreate, PrismaSelect, PrismaUpdate } from "../builders/types";

export class InputNode {
    __typename: `${GqlModelType}`;
    id: string;
    action: QueryAction;
    children: InputNode[];
    parent: InputNode | null;

    constructor(__typename: `${GqlModelType}`, id: string, action: QueryAction) {
        this.__typename = __typename;
        this.id = id;
        this.action = action;
        this.children = [];
        this.parent = null;
    }
}

export type QueryAction = "Connect" | "Create" | "Delete" | "Disconnect" | "Read" | "Update";
export type IdsByAction = { [action in QueryAction]?: string[] };
export type IdsByType = { [objectType in GqlModelType]?: string[] };
export type InputsById = { [id: string]: { node: InputNode, input: unknown } };
export type InputsByType = { [objectType in GqlModelType]?: {
    Connect: { node: InputNode, input: string; }[];
    Create: { node: InputNode, input: PrismaUpdate }[];
    Delete: { node: InputNode, input: string; }[];
    Disconnect: { node: InputNode, input: string; }[];
    Read: { node: InputNode, input: PrismaSelect }[];
    Update: { node: InputNode, input: PrismaUpdate }[];
} };
export type IdsByPlaceholder = { [placeholder: string]: string | null };
export type IdsCreateToConnect = { [id: string]: string };
// TODO add other types later. Reason is that this is used for cudHelper and cudInputsToMaps, which originally only supported the 3 types below. Ideally we want them to support all types, but baby steps :)
export type CudInputData = {
    actionType: "Delete"; //"Connect" | "Delete" | "Disconnect";
    input: string;
    objectType: GqlModelType | `${GqlModelType}`;
} | {
    actionType: "Create";
    input: PrismaCreate;
    objectType: GqlModelType | `${GqlModelType}`;
} | {
    actionType: "Update";
    input: PrismaUpdate;
    objectType: GqlModelType | `${GqlModelType}`;
};
// | {
//     actionType: "Read";
//     input: PrismaSelect;
//     objectType: GqlModelType | `${GqlModelType}`;
// };
