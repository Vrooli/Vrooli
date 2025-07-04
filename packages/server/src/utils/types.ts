import { type ModelType } from "@vrooli/shared";
import { type PrismaCreate, type PrismaSelect, type PrismaUpdate } from "../builders/types.js";
import { type InputNode } from "./inputNode.js";

export type QueryAction = "Connect" | "Create" | "Delete" | "Disconnect" | "Read" | "Update";
export type IdsByAction = { [action in QueryAction]?: string[] };
export type IdsByType = { [objectType in ModelType]?: string[] };
export type InputsById = { [id: string]: { node: InputNode, input: unknown } };
export type InputsByType = { [objectType in ModelType]?: {
    Connect: { node: InputNode, input: string | bigint; }[];
    Create: { node: InputNode, input: PrismaUpdate }[];
    Delete: { node: InputNode, input: string | bigint; }[];
    Disconnect: { node: InputNode, input: string | bigint; }[];
    Read: { node: InputNode, input: PrismaSelect }[];
    Update: { node: InputNode, input: PrismaUpdate }[];
} };
export type IdsByPlaceholder = { [placeholder: string]: string | bigint | null };
export type IdsCreateToConnect = { [id: string]: string };
// TODO add other types later. Reason is that this is used for cudHelper and cudInputsToMaps, which originally only supported the 3 types below. Ideally we want them to support all types, but baby steps :)
export type CudInputData = {
    action: "Delete"; //"Connect" | "Delete" | "Disconnect";
    input: string;
    objectType: ModelType | `${ModelType}`;
} | {
    action: "Create";
    input: PrismaCreate;
    objectType: ModelType | `${ModelType}`;
} | {
    action: "Update";
    input: PrismaUpdate;
    objectType: ModelType | `${ModelType}`;
};
// | {
//     action: "Read";
//     input: PrismaSelect;
//     objectType: ModelType | `${ModelType}`;
// };
export type ResultsById = { [id: string]: unknown };
