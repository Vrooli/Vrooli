import { Report, ReportCreateInput, ReportUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ReportShape = Pick<Report, 'id'>

export const shapeReport: ShapeModel<ReportShape, ReportCreateInput, ReportUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}