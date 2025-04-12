import { Box, styled } from "@mui/material";
import { useFormikContext } from "formik";
import { ReactNode, useMemo } from "react";
import { ViewDisplayType } from "../../types.js";

type OuterFormProps = {
    children: ReactNode;
    display: ViewDisplayType;
    maxWidth?: number | "unset" | "100%";
};

type InnerFormProps = Pick<OuterFormProps, "display" | "maxWidth"> & {
    children: ReactNode;
    isLoading?: boolean;
    /** If true, we'll use a "div" tag instead of "form" */
    isNested?: boolean;
    style?: { [x: string]: string | number | null };
};

type BaseFormProps = InnerFormProps;

const DEFAULT_MAX_WIDTH_PX = 600;

const LoadingOverlay = styled(Box)(() => ({
    position: "absolute",
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    backgroundColor: "rgba(0, 0, 0, 0.2)",
    zIndex: 1,
}));

export function OuterForm({
    children,
    display,
    maxWidth = DEFAULT_MAX_WIDTH_PX,
}: OuterFormProps) {
    const innerStyle = useMemo(function innerStyleMemo() {
        return {
            height: display === "dialog" ? "100%" : "unset",
            overflowY: display === "dialog" ? "auto" : "unset",
            maxWidth: typeof maxWidth === "number" ? `${maxWidth}px` : maxWidth,
            margin: "auto",
            paddingLeft: 1,
            paddingRight: 1,
            paddingBottom: "56px",
        } as const;
    }, [display, maxWidth]);

    return (
        <Box flex={1} style={innerStyle}>
            {children}
        </Box>
    );
}

export function InnerForm({
    children,
    display,
    isLoading = false,
    isNested,
    maxWidth = DEFAULT_MAX_WIDTH_PX,
    style,
}: InnerFormProps) {
    const { handleReset, handleSubmit } = useFormikContext();

    const innerStyle = useMemo(function innerStyleMemo() {
        return {
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "flex-start",
            width: "100%",
            maxWidth: typeof maxWidth === "number" ? `${maxWidth}px` : maxWidth,
            margin: "auto",
            paddingBottom: "64px",
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
            sx={innerStyle}
        >
            {isLoading && <LoadingOverlay />}
            {children}
        </Box>
    );
}

export function BaseForm({
    display,
    maxWidth,
    ...rest
}: BaseFormProps) {
    return (
        <OuterForm display={display} maxWidth={maxWidth}>
            <InnerForm display={display} maxWidth={maxWidth} {...rest} />
        </OuterForm>
    );
} 
