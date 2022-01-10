import { CommonProps } from "types";
import { Forms } from "utils";

export interface FormProps extends Partial<CommonProps> {
    onFormChange?: (form: Forms) => any;
}

export interface LogInFormProps extends FormProps {
    code?: string;
}

export interface ResetPasswordFormProps extends FormProps {
    userId?: string;
    code?: string;
}