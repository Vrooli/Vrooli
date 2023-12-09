import { Form } from "formik";
import { ReactNode } from "react";
import { ViewDisplayType } from "views/types";

export const BaseForm = ({
    children,
    display,
    isLoading = false,
    maxWidth,
    style,
}: {
    display: ViewDisplayType;
    children: ReactNode;
    isLoading?: boolean;
    maxWidth?: number;
    style?: { [x: string]: string | number | null };
}) => {

    return (
        <Form style={{
            display: "block",
            margin: "auto",
            alignItems: "center",
            justifyContent: "center",
            width: maxWidth ? `min(${maxWidth}px, 100vw - 16px)` : "-webkit-fill-available",
            maxWidth: "100%",
            paddingBottom: display === "dialog" ? "16px" : "64px", // Make room for the submit buttons
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
