import { CircularProgress, Dialog, DialogContent, DialogTitle, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TranscriptDialogProps } from "../types";


/**
 * TranscriptDialog - A dialog for displaying the current transcript and listening animation
 */
export const TranscriptDialog = ({ 
    isListening, 
    lng,
    transcript 
}: TranscriptDialogProps) => {
    const { t } = useTranslation();

    return (
        <Dialog open={isListening}>
            <DialogTitle>{t(`common:Listening`, { lng })}</DialogTitle>
            <DialogContent>
                {/* Centered transcript */}
                <Typography align="center" variant="h6">
                    {transcript}
                </Typography>
                {/* <CircularProgress size={60} /> */}
            </DialogContent>
        </Dialog>
    );
};