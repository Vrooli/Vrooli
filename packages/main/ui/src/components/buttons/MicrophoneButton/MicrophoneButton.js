import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { MicrophoneDisabledIcon, MicrophoneOffIcon, MicrophoneOnIcon } from "@local/icons";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useSpeech } from "../../../utils/hooks/useSpeech";
import { PubSub } from "../../../utils/pubsub";
import { TranscriptDialog } from "../../dialogs/TranscriptDialog/TranscriptDialog";
export const MicrophoneButton = ({ disabled = false, onTranscriptChange, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { transcript, isListening, isSpeechSupported, startListening, stopListening, resetTranscript } = useSpeech();
    const status = useMemo(() => {
        if (disabled || !isSpeechSupported)
            return "Disabled";
        if (isListening)
            return "On";
        return "Off";
    }, [disabled, isListening, isSpeechSupported]);
    useEffect(() => {
        if (!isListening)
            onTranscriptChange(transcript);
    }, [isListening, transcript, onTranscriptChange]);
    const Icon = useMemo(() => {
        if (status === "On")
            return MicrophoneOnIcon;
        if (status === "Off")
            return MicrophoneOffIcon;
        return MicrophoneDisabledIcon;
    }, [status]);
    const handleClick = useCallback(() => {
        if (status === "On") {
            stopListening();
            onTranscriptChange(transcript);
        }
        else if (status === "Off") {
            startListening();
            transcript && resetTranscript();
        }
        else {
            PubSub.get().publishSnack({ messageKey: "SpeechNotAvailable", severity: "Error" });
        }
        return true;
    }, [status, startListening, stopListening, transcript, onTranscriptChange, resetTranscript]);
    if (!isSpeechSupported)
        return null;
    return (_jsxs(Box, { children: [_jsx(Tooltip, { title: status !== "Disabled" ? t("SearchByVoice") : "", children: _jsx(IconButton, { onClick: handleClick, sx: {
                        width: "48px",
                        height: "48px",
                    }, "aria-label": "main-search-icon", children: _jsx(Icon, { fill: palette.background.textSecondary }) }) }), _jsx(TranscriptDialog, { handleClose: () => stopListening(), isListening: status === "On", transcript: transcript })] }));
};
//# sourceMappingURL=MicrophoneButton.js.map