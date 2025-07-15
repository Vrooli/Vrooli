// AI_CHECK: TYPE_SAFETY=cudinput-fix | LAST: 2025-07-04 - Fixed CudInputData with discriminated union supporting ModelType|template literals for Create/Update and DeleteType|template literals for Delete
import { type DeleteType, type ModelType } from "@vrooli/shared";
import { type PrismaSelect, type PrismaUpdate } from "../builders/types.js";
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
export type CudInputData = {
    action: "Create" | "Update";
    input: unknown;
    objectType: ModelType | `${ModelType}`;
} | {
    action: "Delete";
    input: unknown;
    objectType: DeleteType | `${DeleteType}`;
};
export type ResultsById = { [id: string]: unknown };
