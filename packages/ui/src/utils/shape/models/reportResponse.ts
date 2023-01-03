import { ReportResponse, ReportResponseCreateInput, ReportResponseUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ReportResponseShape = Pick<ReportResponse, 'id'>

export const shapeReportResponse: ShapeModel<ReportResponseShape, ReportResponseCreateInput, ReportResponseUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}