import { Box, IconButtonProps, DialogTitle as MuiDialogTitle, DialogTitleProps as MuiDialogTitleProps, styled, useTheme } from "@mui/material";
import { forwardRef, useMemo } from "react";
import { useLocation } from "route";
import { useIsLeftHanded } from "../../../hooks/subscriptions.js";
import { CloseIcon } from "../../../icons/common.js";
import { noSelect } from "../../../styles.js";
import { tryOnClose } from "../../../utils/navigation/urlTools.js";
import { Title } from "../../text/Title.js";
import { DialogTitleProps } from "../types.js";

interface StyledTitleContainerProps extends MuiDialogTitleProps {
    isLeftHanded: boolean;
}

const StyledTitleContainer = styled(MuiDialogTitle, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<StyledTitleContainerProps>(({ isLeftHanded, theme }) => ({
    ...noSelect,
    display: "flex",
    flexDirection: isLeftHanded ? "row-reverse" : "row",
    alignItems: "center",
    padding: theme.spacing(0.5),
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    textAlign: "center",
    fontSize: "2rem",
    [theme.breakpoints.down("sm")]: {
        fontSize: "1.5rem",
    },
}));

interface CloseIconButtonProps extends IconButtonProps {
    isLeftHanded: boolean;
}

const CloseIconButton = styled(MuiDialogTitle, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<CloseIconButtonProps>(({ isLeftHanded, theme }) => ({
    display: "flex",
    marginLeft: isLeftHanded ? "0px" : "auto",
    marginRight: isLeftHanded ? "auto" : "0px",
    padding: theme.spacing(1),
}));

export const DialogTitle = forwardRef(({
    below,
    id,
    onClose,
    startComponent,
    ...titleData
}: DialogTitleProps, ref) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            position: "sticky",
            top: 0,
            zIndex: 2,
            ...titleData.sxs?.root,
        } as const;
    }, [palette.primary.contrastText, palette.primary.dark, titleData.sxs]);

    const titleStyle = useMemo(function titleStyleMemo() {
        return {
            stack: {
                ...(isLeftHanded ? { marginRight: "auto" } : { marginLeft: "auto" }),
                padding: 0,
            },
        } as const;
    }, [isLeftHanded]);

    return (
        <Box ref={ref} sx={outerBoxStyle}>
            <StyledTitleContainer
                id={id}
                isLeftHanded={isLeftHanded}
            >
                {startComponent}
                <Title
                    variant="subheader"
                    {...titleData}
                    sxs={titleStyle}
                />
                <CloseIconButton
                    aria-label="close"
                    edge="end"
                    isLeftHanded={isLeftHanded}
                    onClick={() => { tryOnClose(onClose, setLocation); }}
                >
                    <CloseIcon fill={palette.primary.contrastText} />
                </CloseIconButton>
            </StyledTitleContainer>
            {below}
        </Box>
    );
});
DialogTitle.displayName = "DialogTitle";
