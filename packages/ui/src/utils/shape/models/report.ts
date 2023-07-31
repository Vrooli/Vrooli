import { Report, ReportCreateInput, ReportFor, ReportUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { firstString } from "utils/display/stringTools";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type ReportShape = Pick<Report, "id" | "details" | "language" | "reason"> & {
    __typename?: "Report";
    createdFor: { __typename: ReportFor, id: string };
    otherReason?: string | null;
}

export const shapeReport: ShapeModel<ReportShape, ReportCreateInput, ReportUpdateInput> = {
    create: (d) => ({
        createdForConnect: d.createdFor.id,
        createdFor: d.createdFor.__typename,
        reason: firstString(d.otherReason?.trim(), d.reason),
        ...createPrims(d, "id", "details", "language"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        reason: (u.otherReason ?? u.reason) !== o.reason ? (u.otherReason ?? u.reason) : undefined,
        ...updatePrims(o, u, "id", "details", "language"),
    }, a),
};
