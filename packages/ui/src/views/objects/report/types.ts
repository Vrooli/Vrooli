import { Report, ReportFor, ReportShape } from "@local/shared";
import { CrudPropsDialog, FormProps } from "../../../types.js";

type ReportUpsertPropsDialog = Omit<CrudPropsDialog<Report>, "overrideObject"> & {
    createdFor: { __typename: ReportFor, id: string };
};
export type ReportUpsertProps = ReportUpsertPropsDialog;
export type ReportFormProps = FormProps<Report, ReportShape>
