import { CommonProps } from "types";
import { FORMS } from "utils";

export interface FormProps extends Partial<CommonProps> {
    onFormChange?: (form: FORMS) => any;
}

export interface LogInFormProps extends FormProps {
    code?: string;
}

export interface ResetPasswordFormProps extends FormProps {
    userId?: string;
    code?: string;
}