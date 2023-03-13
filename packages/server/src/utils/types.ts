import { GqlModelType } from "@shared/consts";
import { PrismaUpdate } from "../builders/types";

export type QueryAction = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Read' | 'Update';
export type IdsByAction = { [x in QueryAction]?: string[] };
export type IdsByType = { [key in GqlModelType]?: string[] };
export type InputsById = { [key: string]: PrismaUpdate };