import { Box, IconButton, Tooltip, useTheme } from '@mui/material';
import { CameraOpenIcon, MicrophoneDisabledIcon, MicrophoneOffIcon, MicrophoneOnIcon } from '@shared/icons';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { getUserLanguages, useSpeech } from 'utils';
import { CameraButtonProps } from '../types';

/**
 * A microphone icon that can be used to trigger speech recognition
 */
export const CameraButton = ({
    disabled = false,
    onTranscriptChange,
    session,
}: CameraButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // const { transcript, isListening, isCameraSupported, startListening, stopListening, resetTranscript } = useSpeech();

    // const status = useMemo<MicrophoneStatus>(() => {
    //     if (disabled || !isSpeechSupported) return 'Disabled';
    //     if (isListening) return 'On';
    //     return 'Off';
    // }, [disabled, isListening, isSpeechSupported]);

    // const handleClick = useCallback(() => {
    //     if (status === 'On') {
    //         stopListening();
    //         console.log('transcript before onTranscriptChange', transcript);
    //         onTranscriptChange(transcript);
    //     } else if (status === 'Off') {
    //         startListening();
    //         console.log('transcript before resetTranscript', transcript);
    //         transcript && resetTranscript();
    //     }
    //     return true
    // }, [status, startListening, stopListening, transcript, onTranscriptChange, resetTranscript]);

    // if (!isSpeechSupported) return null;
    // return (
    //     <Box>
    //         <Tooltip title={status !== 'Disabled' ? t(`common:SearchByVoice`, { lng: getUserLanguages(session)[0] }) : ''}>
    //             <IconButton
    //                 onClick={handleClick}
    //                 sx={{
    //                     width: '48px',
    //                     height: '48px',
    //                 }} aria-label="main-search-icon"
    //             >
    //                 <CameraOpenIcon fill={palette.background.textSecondary} />
    //             </IconButton>
    //         </Tooltip>
    //     </Box>
    // )
    return {} as any
}