import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
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
import { IconButton, type IconButtonVariant } from "./IconButton.js";
import { type MicrophoneButtonProps } from "./types.js";

type MicrophoneStatus = "On" | "Off" | "Disabled";

const HINT_AFTER_MILLI = 3000;
const DEFAULT_HEIGHT = 48;
const DEFAULT_WIDTH = 48;

interface TranscriptDialogProps {
    handleClose: () => unknown;
    isListening: boolean;
    showHint: boolean;
    transcript: string;
}

// WaveForm component using Tailwind classes
function WaveForm() {
    return (
        <div className="tw-flex tw-justify-center tw-items-center tw-mb-4 tw-h-[50px] tw-gap-2">
            <div className="tw-wave-bar" />
            <div className="tw-wave-bar" />
            <div className="tw-wave-bar" />
            <div className="tw-wave-bar" />
            <div className="tw-wave-bar" />
        </div>
    );
}

const CustomDialog = styled(Dialog)(() => ({
    zIndex: Z_INDEX.Popup,
    "& > .MuiDialog-container": {
        "& > .MuiPaper-root": {
            borderRadius: "16px",
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

    return (
        <CustomDialog
            onClose={handleClose}
            open={isListening}
        >
            <DialogTitle>{t("Listening")}</DialogTitle>
            <DialogContent>
                <WaveForm />
                {/* Centered transcript */}
                <Typography align="center" variant="body1">
                    {transcript.length > 0 ? transcript : showHint ? t("SpeakClearly") : ""}
                </Typography>
            </DialogContent>
        </CustomDialog>
    );
}


/**
 * Extended MicrophoneButton props including variant
 */
export interface ExtendedMicrophoneButtonProps extends MicrophoneButtonProps {
    variant?: IconButtonVariant;
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
    variant = "transparent",
    width = DEFAULT_WIDTH,
}: ExtendedMicrophoneButtonProps) {
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

    // Use custom size if width/height differ from defaults
    const size = width === height ? width : width || DEFAULT_WIDTH;

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
                <span>
                    <IconButton
                        variant={variant}
                        size={size}
                        className={status === "On" ? "tw-microphone-listening tw-animate-microphone-pulse" : undefined}
                        disabled={isMicrophoneDisabled}
                        aria-label="Search by voice"
                        onClick={handleClick}
                    >
                        <Icon
                            decorative
                            fill={status === "On" ? palette.background.default : fill}
                            info={iconInfo}
                        />
                    </IconButton>
                </span>
            </Tooltip>
        </>
    );
}
