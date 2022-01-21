import { Box, Button, Dialog, DialogTitle, Grid, TextField } from '@mui/material';
import { useCallback, useState } from 'react';
import { ReportDialogProps } from '../types';

enum ReportOptions {
    Inappropriate,
    PII,
    Scam,
    Spam,
    Other
}

const ReportReasons = {
    [ReportOptions.Inappropriate]: 'Inappropriate Content',
    [ReportOptions.PII]: 'Includes Personally Identifiable Information (PII)',
    [ReportOptions.Scam]: 'Scam',
    [ReportOptions.Spam]: 'Spam',
    [ReportOptions.Other]: 'Other',
}

export const ReportDialog = ({
    open,
    onClose,
    title = 'Report',
    reportFor,
    forId,
}: ReportDialogProps) => {
    const [reason, setReason] = useState<ReportOptions | undefined>(undefined);
    const [details, setDetails] = useState<string>('');

    const handleDetailsChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setDetails(e.target.value);
    }, []);

    const onSubmit = useCallback(() => {
        //TODO
    }, [reportFor, forId])

    return (
        <Dialog
            onClose={onClose}
            open={open}
            sx={{
                zIndex: 10000,
                width: 'min(500px, 100vw)',
                textAlign: 'center',
                overflow: 'hidden',
            }}
        >
            <DialogTitle sx={{ background: (t) => t.palette.primary.light }}>
                {title}
            </DialogTitle>
            <Box sx={{ padding: 2 }}>
                {/* Text displaying what you are reporting */}
                {/* Reason selector (with other option) */}
                <TextField
                    fullWidth
                    id="details"
                    name="details"
                    label="Details (Optional)"
                    value={details}
                    onChange={handleDetailsChange}
                    helperText="Enter any details you'd like to share about this indcident. DO NOT include any personal information!"
                />
                {/* Details multi-line text field */}
                {/* Action buttons */}
                <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                        <Button fullWidth onClick={onSubmit}>Submit</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button fullWidth onClick={onClose}>Cancel</Button>
                    </Grid>
                </Grid>
            </Box>
        </Dialog>
    )
}