import { Form } from "formik";
import { BaseFormProps } from "forms/types";

export const BaseForm = ({
    children,
    display,
    isLoading = false,
    maxWidth,
    style,
}: BaseFormProps) => {

    return (
        <Form style={{
            display: "block",
            margin: "auto",
            alignItems: "center",
            justifyContent: "center",
            width: maxWidth ? `min(${maxWidth}px, 100vw - ${display === "page" ? "16px" : "64px"})` : "-webkit-fill-available",
            maxWidth: "100%",
            paddingBottom: "64px", // Make room for the submit buttons
            paddingLeft: display === "dialog" ? "env(safe-area-inset-left)" : undefined,
            paddingRight: display === "dialog" ? "env(safe-area-inset-right)" : undefined,
            ...(style ?? {}),
        }}>
            {/* When loading, display a dark overlay */}
            {isLoading && <div style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                height: "100%",
                backgroundColor: "rgba(0, 0, 0, 0.2)",
                zIndex: 1,
            }} />}
            {children}
        </Form>
    );
};
