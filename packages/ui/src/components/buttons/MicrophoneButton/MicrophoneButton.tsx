import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { TranscriptDialog } from "components/dialogs/TranscriptDialog/TranscriptDialog";
import { useSpeech } from "hooks/useSpeech";
import { MicrophoneDisabledIcon, MicrophoneOffIcon, MicrophoneOnIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "utils/pubsub";
import { MicrophoneButtonProps } from "../types";

type MicrophoneStatus = "On" | "Off" | "Disabled";

const HINT_AFTER_MILLI = 3000;

/**
 * A microphone icon that can be used to trigger speech recognition
 */
export const MicrophoneButton = ({
    disabled = false,
    onTranscriptChange,
}: MicrophoneButtonProps) => {
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
            onTranscriptChange(transcript);
        } else if (status === "Off") {
            setShowHint(false);
            resetTranscriptTimeout();
            startListening();
            transcript && resetTranscript();
        } else {
            PubSub.get().publish("snack", { messageKey: "SpeechNotAvailable", severity: "Error" });
        }
        return true;
    }, [status, stopListening, onTranscriptChange, transcript, startListening, resetTranscript, resetTranscriptTimeout]);

    if (!isSpeechSupported) return null;
    return (
        <Box>
            <Tooltip title={status !== "Disabled" ? t("SearchByVoice") : ""}>
                <IconButton
                    onClick={handleClick}
                    sx={{
                        width: "48px",
                        height: "48px",
                    }} aria-label="main-search-icon"
                >
                    <Icon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            {/* When microphone is active, display current translation */}
            <TranscriptDialog
                handleClose={() => stopListening()}
                isListening={status === "On"}
                showHint={showHint}
                transcript={transcript}
            />
        </Box>
    );
};
