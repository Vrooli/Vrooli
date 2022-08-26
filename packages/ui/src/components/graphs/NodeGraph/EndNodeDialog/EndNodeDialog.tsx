/**
 * Prompts user to select which link the new node should be added on
 */
import { EndNodeDialogProps } from '../types';
import { Dialog } from '@mui/material';
import { getTranslation } from 'utils';
import { useEffect, useState } from 'react';
import { useFormik } from 'formik';
import { DialogTitle } from 'components/dialogs';

const titleAria = 'end-node-dialog-title';

export const EndNodeDialog = ({
    handleClose,
    isOpen,
    data,
    language,
    session,
    zIndex,
}: EndNodeDialogProps) => {

    const [changedData, setChangedData] = useState(data);
    useEffect(() => {
        setChangedData(data);
    }, [data]);

    const formik = useFormik({
        initialValues: {
            title: getTranslation(changedData, 'title', [language], false) ?? '',
            wasSuccessful: changedData.data.wasSuccessful,
        },
        // validationSchema,
        onSubmit: (values) => {
            // TODO
        },
    });

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                onClose={handleClose}
                title={formik.values.title}
            />
            <form onSubmit={formik.handleSubmit}>
                {/* TODO */}
            </form>
        </Dialog>
    )
}