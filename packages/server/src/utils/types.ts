import { GqlModelType } from "@local/shared";
import { PrismaUpdate } from "../builders/types";

export type QueryAction = "Connect" | "Create" | "Delete" | "Disconnect" | "Read" | "Update";
export type IdsByAction = { [x in QueryAction]?: string[] };
export type IdsByType = { [key in GqlModelType]?: string[] };
export type InputsByType = { [key in GqlModelType]?: {
    // Connect: string[];
    Create: PrismaUpdate[];
    Delete: string[];
    // Disconnect: string[];
    // Read: PrismaSelect[];
    Update: PrismaUpdate[];
} };
