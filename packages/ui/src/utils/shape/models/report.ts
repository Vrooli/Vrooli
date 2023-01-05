import { Report, ReportCreateInput, ReportUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type ReportShape = Pick<Report, 'id'>

export const shapeReport: ShapeModel<ReportShape, ReportCreateInput, ReportUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}