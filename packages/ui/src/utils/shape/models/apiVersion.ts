import { ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ApiVersionShape = Pick<ApiVersion, 'id'>

export const shapeApiVersion: ShapeModel<ApiVersionShape, ApiVersionCreateInput, ApiVersionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}