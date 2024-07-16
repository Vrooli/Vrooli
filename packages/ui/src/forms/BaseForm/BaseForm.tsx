import { Box, styled } from "@mui/material";
import { useFormikContext } from "formik";
import { ReactNode, useMemo } from "react";
import { ViewDisplayType } from "views/types";

type BaseFormProps = {
    display: ViewDisplayType;
    children: ReactNode;
    isLoading?: boolean;
    /** If true, we'll use a "div" tag instead of "form" */
    isNested?: boolean;
    maxWidth?: number;
    style?: { [x: string]: string | number | null };
};

const LoadingOverlay = styled(Box)(() => ({
    position: "absolute",
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    backgroundColor: "rgba(0, 0, 0, 0.2)",
    zIndex: 1,
}));

export function BaseForm({
    children,
    display,
    isLoading = false,
    isNested,
    maxWidth,
    style,
}: BaseFormProps) {
    const { handleReset, handleSubmit } = useFormikContext();

    const outerStyle = useMemo(function outerStyleMemo() {
        return {
            display: "block",
            margin: "auto",
            alignItems: "center",
            justifyContent: "center",
            width: maxWidth ? `min(${maxWidth}px, 100vw)` : "-webkit-fill-available",
            maxWidth: "100%",
            paddingBottom: display === "dialog" ? "16px" : "64px", // Make room for the submit buttons
            paddingLeft: display === "dialog" ? "env(safe-area-inset-left)" : undefined,
            paddingRight: display === "dialog" ? "env(safe-area-inset-right)" : undefined,
            ...style,
        } as const;
    }, [display, maxWidth, style]);

    return (
        // iOS needs an "action" attribute to allow inputs to set the virtual keyboard's "Enter" button text: https://stackoverflow.com/a/39485162/406725
        // We default the action to "#" in case the preventDefault fails (just updates the URL hash)
        <Box
            action="#"
            component={isNested ? "div" : "form"}
            onSubmit={handleSubmit as (ev: React.FormEvent<HTMLFormElement | HTMLDivElement>) => void}
            onReset={handleReset}
            sx={outerStyle}
        >
            {isLoading && <LoadingOverlay />}
            {children}
        </Box>
    );
}
