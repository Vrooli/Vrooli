import { Box, BoxProps, List, Typography, styled } from "@mui/material";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";
import { SxType } from "../../types.js";
import { ListContainerProps } from "./types.js";

interface OuterBoxProps extends BoxProps {
    borderRadius?: number;
    isEmpty: boolean;
    sx?: SxType;
}

const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "borderRadius" && prop !== "isEmpty" && prop !== "sx",
})<OuterBoxProps>(({ borderRadius, isEmpty, theme, sx }) => ({
    maxWidth: "1000px",
    width: "100%",
    marginLeft: "auto",
    marginRight: "auto",
    ...(isEmpty ? {} : {
        background: theme.palette.background.paper,
        borderRadius: borderRadius !== undefined ? `${borderRadius}px` : theme.spacing(1),
        overflow: "overlay",
        display: "block",
    }),
    [theme.breakpoints.down("sm")]: {
        borderRadius: 0,
    },
    ...sx,
} as any));

const listStyle = { padding: 0 } as const;

export const ListContainer = forwardRef<HTMLDivElement, ListContainerProps>(({
    borderRadius,
    children,
    emptyText,
    id,
    isEmpty = false,
    sx,
}, ref) => {
    const { t } = useTranslation();

    return (
        <OuterBox
            borderRadius={borderRadius}
            id={id}
            isEmpty={isEmpty}
            ref={ref}
            sx={sx}
        >
            {isEmpty && (
                <Typography variant="body1" color="text.secondary" textAlign="center" pt={2}>
                    {emptyText ?? t("NoResults", { ns: "error" })}
                </Typography>
            )}
            {!isEmpty && (
                <List sx={listStyle}>
                    {children}
                </List>
            )}
        </OuterBox>
    );
});
ListContainer.displayName = "ListContainer";
