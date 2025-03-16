import { API_CREDITS_MULTIPLIER, API_CREDITS_PREMIUM, LINKS } from "@local/shared";
import { Box, Button, Dialog, DialogContent, IconButton, Typography, keyframes, styled } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CloseIcon } from "../../../icons/common.js";
import { useLocation } from "../../../route/router.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { BreadcrumbsBase } from "../../breadcrumbs/BreadcrumbsBase/BreadcrumbsBase.js";

type TriangleProps = {
    size?: number;
    direction?: "up" | "down" | "left" | "right";
    gradient?: readonly [string, string];
    position?: Record<string, string | number>;
    style?: Record<string, string | number>;
};

const DEFAULT_TRIANGLE_SIZE = 100;
const DEFAULT_TRIANGLE_DIRECTION = "up" as const;
const DEFAULT_TRIANGLE_GRADIENT = ["#ffffff", "#000000"] as const;
const DEFAULT_TRIANGLE_POSITION = {};
const DEFAULT_TRIANGLE_STYLE = {};

function Triangle({
    size = DEFAULT_TRIANGLE_SIZE,
    direction = DEFAULT_TRIANGLE_DIRECTION,
    gradient = DEFAULT_TRIANGLE_GRADIENT,
    position = DEFAULT_TRIANGLE_POSITION,
    style = DEFAULT_TRIANGLE_STYLE,
}: TriangleProps) {
    const width = size;
    const height = size;

    // Determine rotation based on direction
    let rotation = 0;
    switch (direction) {
        case "up":
            rotation = 0;
            break;
        case "down":
            // eslint-disable-next-line no-magic-numbers
            rotation = 180;
            break;
        case "left":
            // eslint-disable-next-line no-magic-numbers    
            rotation = -90;
            break;
        case "right":
            // eslint-disable-next-line no-magic-numbers   
            rotation = 90;
            break;
        default:
            rotation = 0;
    }

    // Unique ID for the gradient to avoid conflicts
    const gradientId = `triangle-gradient-${Math.random()}`;

    const svgStyle = useMemo(function svgStyleMemo() {
        return {
            transform: `rotate(${rotation}deg)`,
        } as const;
    }, [rotation]);

    return (
        <Box position="absolute" {...position} style={style}>
            <svg
                width={width}
                height={height}
                style={svgStyle}
            >
                <defs>
                    <linearGradient id={gradientId}>
                        <stop offset="0%" stopColor={gradient[0]} />
                        <stop offset="100%" stopColor={gradient[1]} />
                    </linearGradient>
                </defs>
                <polygon
                    points={`${width / 2},0 0,${height} ${width},${height}`}
                    fill={`url(#${gradientId})`}
                />
            </svg>
        </Box>
    );
}

const StyledDialog = styled(Dialog)(({ theme }) => ({
    "& .MuiPaper-root": {
        borderRadius: 16,
        overflow: "hidden",
        background: "linear-gradient(to bottom right, #565656, #809ba2)",
        position: "relative",
        maxWidth: "28rem",
        margin: theme.spacing(2),
    },
    "& .MuiBackdrop-root": {
        backdropFilter: "blur(4px)",
    },
}));

const CloseButton = styled(IconButton)(({ theme }) => ({
    position: "absolute",
    right: theme.spacing(2),
    top: theme.spacing(2),
    color: theme.palette.text.secondary,
    zIndex: 1,
    transition: "opacity 0.2s",
    "&:hover": {
        opacity: 0.7,
    },
}));

const ContentContainer = styled(Box)(({ theme }) => ({
    // eslint-disable-next-line no-magic-numbers
    padding: theme.spacing(3, 4),
    textAlign: "center",
}));

const TitleText = styled(Typography)(({ theme }) => ({
    color: "white",
    fontWeight: "bold",
    marginBottom: theme.spacing(2),
    textShadow:
        `-2px 2px 0 #333333,
            2px 2px 0 #333333`,
}));

const SubtitleText = styled(Typography)(() => ({
    color: "#ffffffb3",
    lineHeight: 1.6,
}));

const ButtonContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    // eslint-disable-next-line no-magic-numbers
    gap: theme.spacing(1.5),
    // eslint-disable-next-line no-magic-numbers
    marginTop: theme.spacing(6),
}));

const pulse = keyframes`
    0% {
        box-shadow: 0 0 0 0 rgba(166, 92, 179, 0.7);
    }
    70% {
        box-shadow: 0 0 0 10px rgba(166, 92, 179, 0);
    }
    100% {
        box-shadow: 0 0 0 0 rgba(166, 92, 179, 0);
    }
`;
const upgradeButtonBackground = "linear-gradient(to right, #944b9e, #5cd9e8)" as const;
const UpgradeButton = styled(Button)(({ theme }) => ({
    background: upgradeButtonBackground,
    color: "white",
    // eslint-disable-next-line no-magic-numbers
    padding: theme.spacing(1.5),
    animation: `${pulse} 3s infinite ease`,
    "&:hover": {
        background: upgradeButtonBackground,
        filter: "brightness(1.2)",
    },
}));

const CreditsButton = styled(Button)(() => ({
    border: "1px solid #4372a3",
    color: "white",
    transition: "background 0s",
    "&:hover": {
        background: "#4372a3",
        filter: "brightness(1.2)",
    },
}));

const largeTriangeProps = {
    size: 700,
    direction: "down",
    gradient: ["#253f97cc", "#283540cc"],
    position: { bottom: "-575px", left: "-400px" },
    style: { transform: "rotate(50deg)" },
} as const;
const smallTriangleProps = {
    size: 1000,
    direction: "up",
    gradient: ["#191637cc", "#25458cc5"],
    position: { bottom: "-525px", left: "-100px" },
    style: { transform: "rotate(40deg)" },
} as const;

const breadcrumbsStyle = {
    margin: "auto",
    zIndex: 1,
} as const;

export interface ProDialogProps {
    isOpen: boolean;
    onClose: () => unknown;
}

export function ProDialog({ isOpen, onClose }: ProDialogProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const breadcrumbPaths = [
        {
            text: t("Feature", { count: 2 }),
            link: `${LINKS.Pro}#${ELEMENT_IDS.ProViewFeatures}`,
        },
        {
            text: t("Donate"),
            link: `${LINKS.Pro}#${ELEMENT_IDS.ProViewDonateBox}`,
        },
        {
            text: t("Faq"),
            link: `${LINKS.Pro}#${ELEMENT_IDS.ProViewFAQBox}`,
        },
        {
            text: t("Terms"),
            link: LINKS.Terms,
        },
    ] as const;

    const openUpgrade = useCallback(function openUpgradeCallback() {
        setLocation(`${LINKS.Pro}#${ELEMENT_IDS.ProViewSubscribeBox}`);
        onClose();
    }, [onClose, setLocation]);
    const openBuyCredits = useCallback(function openBuyCreditsCallback() {
        setLocation(`${LINKS.Pro}#${ELEMENT_IDS.ProViewBuyCreditsBox}`);
        onClose();
    }, [onClose, setLocation]);

    return (
        <StyledDialog
            open={isOpen}
            onClose={onClose}
        >
            <CloseButton onClick={onClose}>
                <CloseIcon />
            </CloseButton>
            <Triangle {...largeTriangeProps} />
            <Triangle {...smallTriangleProps} />
            <DialogContent>
                <ContentContainer>
                    <TitleText variant="h4">
                        Unlock Pro Features
                    </TitleText>

                    <SubtitleText variant="body1">
                        Upgrade to access bots, run routines, and more. Pro users get ${(Number(API_CREDITS_PREMIUM / API_CREDITS_MULTIPLIER) / 100).toFixed(0)} in monthly credits, increased limits, and take priority in scheduled runs.
                    </SubtitleText>

                    <ButtonContainer>
                        <UpgradeButton
                            variant="contained"
                            size="large"
                            onClick={openUpgrade}
                        >
                            Upgrade Now
                        </UpgradeButton>

                        <CreditsButton
                            variant="outlined"
                            size="large"
                            onClick={openBuyCredits}
                        >
                            Buy Credits
                        </CreditsButton>

                        <BreadcrumbsBase
                            onClick={onClose}
                            paths={breadcrumbPaths}
                            separator={"•"}
                            sx={breadcrumbsStyle}
                        />
                    </ButtonContainer>
                </ContentContainer>
            </DialogContent>
        </StyledDialog>
    );
}
