import { type Report, type ReportFor, type ReportShape } from "@local/shared";
import { type CrudPropsDialog, type FormProps } from "../../../types.js";

type ReportUpsertPropsDialog = Omit<CrudPropsDialog<Report>, "overrideObject"> & {
    createdFor: { __typename: ReportFor, id: string };
};
export type ReportUpsertProps = ReportUpsertPropsDialog;
export type ReportFormProps = FormProps<Report, ReportShape>
