import { GqlModelType } from "@local/shared";
import { PrismaCreate, PrismaSelect, PrismaUpdate } from "../builders/types";
import { InputNode } from "./inputNode";

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
    action: "Delete"; //"Connect" | "Delete" | "Disconnect";
    input: string;
    objectType: GqlModelType | `${GqlModelType}`;
} | {
    action: "Create";
    input: PrismaCreate;
    objectType: GqlModelType | `${GqlModelType}`;
} | {
    action: "Update";
    input: PrismaUpdate;
    objectType: GqlModelType | `${GqlModelType}`;
};
// | {
//     action: "Read";
//     input: PrismaSelect;
//     objectType: GqlModelType | `${GqlModelType}`;
// };
