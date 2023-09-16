import { GqlModelType } from "@local/shared";
import { PrismaSelect, PrismaUpdate } from "../builders/types";

export class InputNode {
    __typename: string;
    id: string;
    action: QueryAction;
    children: InputNode[];
    parent: InputNode | null;

    constructor(__typename: string, id: string, action: QueryAction) {
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
