import { useTheme } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { CameraButtonProps } from '../types';

/**
 * A microphone icon that can be used to trigger speech recognition
 */
export const CameraButton = ({
    disabled = false,
    onTranscriptChange,
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
    //         <Tooltip title={status !== 'Disabled' ? t(`SearchByVoice`) : ''}>
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