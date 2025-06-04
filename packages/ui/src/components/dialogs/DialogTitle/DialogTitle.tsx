import { Box, type IconButtonProps, DialogTitle as MuiDialogTitle, type DialogTitleProps as MuiDialogTitleProps, styled, useTheme } from "@mui/material";
import { forwardRef, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useIsLeftHanded } from "../../../hooks/subscriptions.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { noSelect } from "../../../styles.js";
import { tryOnClose } from "../../../utils/navigation/urlTools.js";
import { Title } from "../../text/Title.js";
import { type DialogTitleProps } from "../types.js";

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
    justifyContent: "space-between",
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
const titleStyle = { stack: { padding: 0 } } as const;

export const DialogTitle = forwardRef(({
    below,
    id,
    onClose,
    startComponent,
    ...titleData
}: DialogTitleProps, ref) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();

    function handleClose() {
        tryOnClose(onClose, setLocation);
    }

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
                    aria-label={t("Close")}
                    edge="end"
                    isLeftHanded={isLeftHanded}
                    onClick={handleClose}
                >
                    <IconCommon
                        decorative
                        fill={palette.primary.contrastText}
                        name="Close"
                    />
                </CloseIconButton>
            </StyledTitleContainer>
            {below}
        </Box>
    );
});
DialogTitle.displayName = "DialogTitle";
