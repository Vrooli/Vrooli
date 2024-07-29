import { Report, ReportShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps } from "../../../types";
import { NewReportShape } from "./ReportUpsert/ReportUpsert";

type ReportUpsertPropsPage = CrudPropsPage;
type ReportUpsertPropsDialog = Omit<CrudPropsDialog<Report>, "overrideObject"> & {
    overrideObject?: NewReportShape;
};
export type ReportUpsertProps = ReportUpsertPropsPage | ReportUpsertPropsDialog;
export type ReportFormProps = FormProps<Report, ReportShape>
