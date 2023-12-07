import { ReportResponse, ReportResponseCreateInput, ReportResponseUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type ReportResponseShape = Pick<ReportResponse, "id"> & {
    __typename: "ReportResponse";
}

export const shapeReportResponse: ShapeModel<ReportResponseShape, ReportResponseCreateInput, ReportResponseUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any,
};
