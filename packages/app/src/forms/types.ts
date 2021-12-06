import { CommonProps } from "types";
import { FORMS } from "utils";

export interface FormProps extends Partial<CommonProps> {
    onFormChange?: (form: FORMS) => any;
}