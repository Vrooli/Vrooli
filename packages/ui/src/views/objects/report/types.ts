import { Report } from "@local/shared";
import { FormProps } from "forms/types";
import { ReportShape } from "utils/shape/models/report";
import { UpsertProps } from "../types";
import { NewReportShape } from "./ReportUpsert/ReportUpsert";

export type ReportUpsertProps = Omit<UpsertProps<Report>, "overrideObject"> & {
    overrideObject?: NewReportShape;
}
export type ReportFormProps = FormProps<Report, ReportShape>
