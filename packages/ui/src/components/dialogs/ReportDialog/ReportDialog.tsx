import { useMutation } from '@apollo/client';
import { reportCreate as validationSchema } from '@local/shared';
import { Autocomplete, Box, Button, Dialog, Grid, IconButton, Stack, TextField, Typography } from '@mui/material';
import { HelpButton } from 'components/buttons';
import { useFormik } from 'formik';
import { reportCreate, reportCreateVariables } from 'graphql/generated/reportCreate';
import { reportCreateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useCallback, useState } from 'react';
import { ReportDialogProps } from '../types';
import { 
    Close as CloseIcon
} from '@mui/icons-material';
import { Pubs } from 'utils';

const helpText =
    `
# Header1 test
## Header2 test
**bold test**
fkdjslakfd;

fdsjlakfjdl;k
`

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

    const [mutation, { loading }] = useMutation<reportCreate>(reportCreateMutation);
    const formik = useFormik({
        initialValues: {
            reason: '',
            details: ''
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: values,
                successCondition: (response) => response.data.emailLogIn !== null,
                onSuccess: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Report submitted.' });
                    onClose() 
                },
            })
        },
    });


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
            <Box sx={{
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
            }}>
                <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                    {title}
                </Typography>
                <Box sx={{ marginLeft: 'auto' }}>
                    <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
                    <IconButton
                        edge="start"
                        onClick={onClose}
                    >
                        <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                    </IconButton>
                </Box>
            </Box>
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