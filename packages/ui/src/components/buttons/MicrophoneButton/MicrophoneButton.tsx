import { Box, Dialog, DialogContent, DialogTitle, IconButton, IconButtonProps, Palette, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSpeech } from "../../../hooks/useSpeech.js";
import { MicrophoneDisabledIcon, MicrophoneOffIcon, MicrophoneOnIcon } from "../../../icons/common.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { MicrophoneButtonProps } from "../types.js";

type MicrophoneStatus = "On" | "Off" | "Disabled";

const HINT_AFTER_MILLI = 3000;
const DEFAULT_HEIGHT = 48;
const DEFAULT_WIDTH = 48;

type StyledIconButtonProps = IconButtonProps & {
    disabled?: boolean;
    height?: number;
    status: MicrophoneStatus;
    width?: number;
};
const StyledIconButton = styled(IconButton)<StyledIconButtonProps>(({ theme, disabled, height, status, width }) => ({
    width: width || DEFAULT_WIDTH,
    height: height || DEFAULT_HEIGHT,
    padding: (width && width <= 16) ? "0px" : (width && width <= 32) ? "4px" : "8px",
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

    const Icon = useMemo(() => {
        if (status === "On") return MicrophoneOnIcon;
        if (status === "Off") return MicrophoneOffIcon;
        return MicrophoneDisabledIcon;
    }, [status]);

    const handleClick = useCallback(() => {
        if (status === "On") {
            stopListening();
        } else if (status === "Off") {
            setShowHint(false);
            resetTranscriptTimeout();
            startListening();
            transcript && resetTranscript();
        } else {
            PubSub.get().publish("snack", { messageKey: "SpeechNotAvailable", severity: "Error" });
        }
        return true;
    }, [status, stopListening, transcript, startListening, resetTranscript, resetTranscriptTimeout]);

    if (!isSpeechSupported && !showWhenUnavailable) return null;
    return (
        <Box>
            <Tooltip title={status !== "Disabled" ? t("SearchByVoice") : ""}>
                <StyledIconButton
                    height={height}
                    onClick={handleClick}
                    status={status}
                    width={width}
                >
                    <Icon
                        fill={status === "On" ? palette.background.default : (fill ?? palette.background.textPrimary)}
                        width="100%"
                        height="100%"
                    />
                </StyledIconButton>
            </Tooltip>
            {/* When microphone is active, display current translation */}
            <TranscriptDialog
                handleClose={stopListening}
                isListening={status === "On"}
                showHint={showHint}
                transcript={transcript}
            />
        </Box>
    );
}
