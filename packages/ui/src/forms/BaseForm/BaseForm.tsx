import { Box, styled } from "@mui/material";
import { useFormikContext } from "formik";
import { forwardRef, ReactNode, useMemo } from "react";
import { ViewDisplayType } from "../../types.js";

type OuterFormProps = {
    children: ReactNode;
    display: ViewDisplayType | `${ViewDisplayType}`;
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

export const OuterForm = forwardRef<HTMLDivElement, OuterFormProps>(({
    children,
    display,
    maxWidth = DEFAULT_MAX_WIDTH_PX,
}, ref) => {
    const innerStyle = useMemo(function innerStyleMemo() {
        return {
            height: display === ViewDisplayType.Dialog ? "100%" : "unset",
            overflowY: display === ViewDisplayType.Dialog ? "auto" : "unset",
            maxWidth: typeof maxWidth === "number" ? `${maxWidth}px` : maxWidth,
            margin: "auto",
            paddingLeft: 1,
            paddingRight: 1,
            paddingBottom: "56px",
        } as const;
    }, [display, maxWidth]);

    return (
        <Box flex={1} style={innerStyle} ref={ref}>
            {children}
        </Box>
    );
});
OuterForm.displayName = 'OuterForm';

export const InnerForm = forwardRef<HTMLDivElement, InnerFormProps>(({
    children,
    display,
    isLoading = false,
    isNested,
    maxWidth = DEFAULT_MAX_WIDTH_PX,
    style,
}, ref) => {
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
            paddingLeft: display === ViewDisplayType.Dialog ? "env(safe-area-inset-left)" : undefined,
            paddingRight: display === ViewDisplayType.Dialog ? "env(safe-area-inset-right)" : undefined,
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
            ref={ref}
        >
            {isLoading && <LoadingOverlay />}
            {children}
        </Box>
    );
});
InnerForm.displayName = 'InnerForm';

export const BaseForm = forwardRef<HTMLDivElement, BaseFormProps>(({
    display,
    maxWidth,
    ...rest
}, ref) => {
    return (
        <OuterForm display={display} maxWidth={maxWidth} ref={ref}>
            <InnerForm display={display} maxWidth={maxWidth} {...rest} />
        </OuterForm>
    );
});
BaseForm.displayName = 'BaseForm';
