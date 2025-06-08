import Box from "@mui/material/Box";
import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import IconButton from "@mui/material/IconButton";
import type { IconButtonProps } from "@mui/material/IconButton";
import type { Palette } from "@mui/material/styles";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { styled, useTheme } from "@mui/material/styles";
import i18next from "i18next";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSpeech } from "../../hooks/useSpeech.js";
import { Icon } from "../../icons/Icons.js";
import { Z_INDEX } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { type MicrophoneButtonProps } from "./types.js";

type MicrophoneStatus = "On" | "Off" | "Disabled";

const HINT_AFTER_MILLI = 3000;
const DEFAULT_HEIGHT = 48;
const DEFAULT_WIDTH = 48;
const PADDING_SMALL_THRESHOLD_PX = 16;
const PADDING_MEDIUM_THRESHOLD_PX = 32;

type StyledIconButtonProps = IconButtonProps & {
    disabled?: boolean;
    height?: number;
    status: MicrophoneStatus;
    width?: number;
};
const StyledIconButton = styled(IconButton)<StyledIconButtonProps>(({ theme, disabled, height, status, width }) => ({
    width: width || DEFAULT_WIDTH,
    height: height || DEFAULT_HEIGHT,
    padding: (width && width <= PADDING_SMALL_THRESHOLD_PX) ? "0px" : (width && width <= PADDING_MEDIUM_THRESHOLD_PX) ? "4px" : "8px",
    background: status === "On" ? theme.palette.background.textSecondary : "transparent",
    opacity: disabled ? 0.5 : 1,
}));

interface TranscriptDialogProps {
    handleClose: () => unknown;
    isListening: boolean;
    showHint: boolean;
    transcript: string;
}

function waveBarSx(palette: Palette) {
    return {
        width: "8px",
        background: `linear-gradient(to top, ${palette.secondary.dark}, ${palette.secondary.light})`,
        animation: "waveAnimation 1s infinite ease-in-out",
        borderRadius: "5px",
        "&:nth-child(2)": { animationDelay: "0.1s" },
        "&:nth-child(3)": { animationDelay: "0.2s" },
        "&:nth-child(4)": { animationDelay: "0.3s" },
        "&:nth-child(5)": { animationDelay: "0.4s" },
    };
}

const WaveForm = styled(Box)(() => ({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    marginBottom: "16px",
    height: "50px",
    gap: "8px",
    "@keyframes waveAnimation": {
        "0%, 40%, 100%": { height: "20px" },
        "20%": { height: "50px" },
    },
}));

const CustomDialog = styled(Dialog)(({ theme }) => ({
    zIndex: Z_INDEX.Popup,
    "& > .MuiDialog-container": {
        "& > .MuiPaper-root": {
            borderRadius: theme.spacing(2),
            zIndex: Z_INDEX.Popup,
        },
    },
}));

/**
 * TranscriptDialog - A dialog for displaying the current transcript and listening animation
 */
export function TranscriptDialog({
    handleClose,
    isListening,
    showHint,
    transcript,
}: TranscriptDialogProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    return (
        <CustomDialog
            onClose={handleClose}
            open={isListening}
        >
            <DialogTitle>{t("Listening")}</DialogTitle>
            <DialogContent>
                <WaveForm>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                </WaveForm>
                {/* Centered transcript */}
                <Typography align="center" variant="body1">
                    {transcript.length > 0 ? transcript : showHint ? t("SpeakClearly") : ""}
                </Typography>
            </DialogContent>
        </CustomDialog>
    );
}


/**
 * A microphone icon that can be used to trigger speech recognition
 */
export function MicrophoneButton({
    disabled = false,
    fill,
    height = DEFAULT_HEIGHT,
    onTranscriptChange,
    showWhenUnavailable = false,
    width = DEFAULT_WIDTH,
}: MicrophoneButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { transcript, isListening, isSpeechSupported, startListening, stopListening, resetTranscript } = useSpeech();
    const [transcriptTimeout, setTranscriptTimeout] = useState<NodeJS.Timeout | null>(null);
    const [showHint, setShowHint] = useState(false);

    const status = useMemo<MicrophoneStatus>(() => {
        if (disabled || !isSpeechSupported) return "Disabled";
        if (isListening) return "On";
        return "Off";
    }, [disabled, isListening, isSpeechSupported]);

    const hasSentTranscript = useRef(false);
    useEffect(function sendTranscriptChangeEffect() {
        if (transcript && !isListening && !hasSentTranscript.current) {
            onTranscriptChange(transcript);
            hasSentTranscript.current = true;
        }
    }, [transcript, onTranscriptChange, isListening]);
    useEffect(function resetHasSentTranscriptEffect() {
        if (isListening) {
            hasSentTranscript.current = false;
        }
    }, [isListening]);

    const resetTranscriptTimeout = useCallback(() => {
        // Clear previous timeout
        if (transcriptTimeout) {
            clearTimeout(transcriptTimeout);
        }
        // Set a new timeout
        setTranscriptTimeout(setTimeout(() => {
            if (!transcript) {
                setShowHint(true);
            }
        }, HINT_AFTER_MILLI));
    }, [transcriptTimeout, transcript]);

    const iconInfo = useMemo(() => {
        if (status === "On") return { name: "MicrophoneOn", type: "Common" } as const;
        if (status === "Off") return { name: "MicrophoneOff", type: "Common" } as const;
        return { name: "MicrophoneDisabled", type: "Common" } as const;
    }, [status]);
    const isMicrophoneDisabled = status === "Disabled";

    const handleClick = useCallback(() => {
        if (status === "On") {
            stopListening();
        } else if (status === "Off") {
            setShowHint(false);
            resetTranscriptTimeout();
            startListening();
            transcript && resetTranscript();
        } else {
            PubSub.get().publish("snack", { message: i18next.t("SpeechNotAvailable"), severity: "Error" });
        }
        return true;
    }, [status, stopListening, transcript, startListening, resetTranscript, resetTranscriptTimeout]);

    if (!isSpeechSupported && !showWhenUnavailable) return null;
    return (
        <>
            <TranscriptDialog
                handleClose={stopListening}
                isListening={status === "On"}
                showHint={showHint}
                transcript={transcript}
            />
            <Tooltip title={isMicrophoneDisabled ? t("SpeechNotAvailable") : t("SearchByVoice")}>
                <StyledIconButton
                    aria-disabled={isMicrophoneDisabled}
                    aria-label="Search by voice"
                    height={height}
                    onClick={handleClick}
                    status={status}
                    width={width}
                >
                    <Icon
                        decorative
                        fill={status === "On" ? palette.background.default : (fill ?? palette.background.textPrimary)}
                        info={iconInfo}
                    />
                </StyledIconButton>
            </Tooltip>
        </>
    );
}
