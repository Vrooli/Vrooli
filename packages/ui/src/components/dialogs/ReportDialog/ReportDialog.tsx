import { useMutation } from '@apollo/client';
import { Autocomplete, Button, Dialog, DialogTitle, Grid, Stack, TextField } from '@mui/material';
import { reportCreate_reportCreate } from 'graphql/generated/reportCreate';
import { reportCreateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
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
    const [reason, setReason] = useState<string | undefined>(undefined);
    const [details, setDetails] = useState<string>('');

    console.log('reason', reason);

    const handleReasonChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setReason(e.target.value);
    }, []);
    const handleDetailsChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setDetails(e.target.value);
    }, []);

    const [mutation] = useMutation<reportCreate_reportCreate>(reportCreateMutation);
    const onSubmit = useCallback(() => {
        mutationWrapper({
            mutation,
            input: {
                createdFor: reportFor as any,
                createdForId: forId,
                details,
                reason,
            },
        })
    }, [details, reason, reportFor, forId]);

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
            <Stack direction="column" spacing={2} sx={{ padding: 2 }}>
                {/* Text displaying what you are reporting */}
                <Autocomplete
                    disablePortal
                    id="report-reason"
                    options={Object.entries(ReportReasons)}
                    inputValue={reason}
                    getOptionLabel={(option) => option[1]}
                    renderInput={(params) => (
                        <TextField
                            fullWidth
                            id="report-reason-input"
                            name="reason"
                            label="Reason"
                            value={reason}
                            onChange={handleReasonChange}
                            helperText="Select or enter the reason you're submitting this report..."
                        />
                    )}
                />
                {/* Reason selector (with other option) */}
                <TextField
                    fullWidth
                    id="report-details"
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
            </Stack>
        </Dialog>
    )
}