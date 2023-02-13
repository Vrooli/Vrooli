import { Box, IconButton, Tooltip, useTheme } from '@mui/material';
import { MicrophoneDisabledIcon, MicrophoneOffIcon, MicrophoneOnIcon } from '@shared/icons';
import { TranscriptDialog } from 'components/dialogs';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { getUserLanguages, useSpeech } from 'utils';
import { MicrophoneButtonProps } from '../types';

type MicrophoneStatus = 'On' | 'Off' | 'Disabled';

/**
 * A microphone icon that can be used to trigger speech recognition
 */
export const MicrophoneButton = ({
    disabled = false,
    onTranscriptChange,
    session,
}: MicrophoneButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { transcript, isListening, isSpeechSupported, startListening, stopListening, resetTranscript } = useSpeech();

    const status = useMemo<MicrophoneStatus>(() => {
        if (disabled || !isSpeechSupported) return 'Disabled';
        if (isListening) return 'On';
        return 'Off';
    }, [disabled, isListening, isSpeechSupported]);

    const Icon = useMemo(() => {
        if (status === 'On') return MicrophoneOnIcon;
        if (status === 'Off') return MicrophoneOffIcon;
        return MicrophoneDisabledIcon;
    }, [status]);

    const handleClick = useCallback(() => {
        if (status === 'On') {
            stopListening();
            console.log('transcript before onTranscriptChange', transcript);
            onTranscriptChange(transcript);
        } else if (status === 'Off') {
            startListening();
            console.log('transcript before resetTranscript', transcript);
            transcript && resetTranscript();
        }
        return true
    }, [status, startListening, stopListening, transcript, onTranscriptChange, resetTranscript]);

    if (!isSpeechSupported) return null;
    return (
        <Box>
            <Tooltip title={status !== 'Disabled' ? t(`common:SearchByVoice`, { lng: getUserLanguages(session)[0] }) : ''}>
                <IconButton
                    onClick={handleClick}
                    sx={{
                        width: '48px',
                        height: '48px',
                    }} aria-label="main-search-icon"
                >
                    <Icon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            {/* When microphone is active, display current translation */}
            <TranscriptDialog
                handleClose={() => stopListening()}
                isListening={status === 'On'}
                lng={getUserLanguages(session)[0]}
                transcript={transcript}
            />
        </Box>
    )
}