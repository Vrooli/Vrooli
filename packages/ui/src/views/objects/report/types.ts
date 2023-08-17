import { Report } from "@local/shared";
import { NewReportShape } from "forms/ReportForm/ReportForm";
import { UpsertProps } from "../types";

export type ReportUpsertProps = Omit<UpsertProps<Report>, "overrideObject"> & {
    overrideObject?: NewReportShape;
}
