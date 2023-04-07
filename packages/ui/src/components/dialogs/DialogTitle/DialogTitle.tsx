import {
    Box,
    DialogTitle as MuiDialogTitle,
    IconButton,
    useTheme
} from '@mui/material';
import { CloseIcon } from '@shared/icons';
import { HelpButton } from 'components/buttons/HelpButton/HelpButton';
import { noSelect } from 'styles';
import { DialogTitleProps } from '../types';

export const DialogTitle = ({
    below,
    helpText,
    id,
    onClose,
    title,
}: DialogTitleProps) => {
    const { palette } = useTheme();

    return (
        <Box sx={{
            background: palette.primary.dark,
            color: palette.primary.contrastText,
        }}>
            <MuiDialogTitle
                id={id}
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 2,
                    textAlign: 'center',
                    fontSize: { xs: '1.5rem', sm: '2rem' },
                }}
            >
                <Box sx={{ marginLeft: 'auto' }} >{title}</Box>
                {helpText && <HelpButton
                    markdown={helpText}
                    sx={{
                        fill: palette.secondary.light,
                    }}
                    sxRoot={{
                        display: 'flex',
                        marginTop: 'auto',
                        marginBottom: 'auto',
                    }}
                />}
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={onClose}
                    sx={{ marginLeft: 'auto' }}
                >
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </MuiDialogTitle>
            {below}
        </Box>
    )
}