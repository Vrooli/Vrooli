import {
    DialogTitle as MuiDialogTitle,
    IconButton,
    Typography,
    useTheme
} from '@mui/material';
import { CloseIcon } from '@shared/icons';
import { HelpButton } from 'components';
import { noSelect } from 'styles';
import { DialogTitleProps } from '../types';

export const DialogTitle = ({
    ariaLabel,
    helpText,
    onClose,
    title,
}: DialogTitleProps) => {
    const { palette } = useTheme();

    return (
        <MuiDialogTitle
            id={ariaLabel}
            sx={{
                ...noSelect,
                display: 'flex',
                alignItems: 'center',
                padding: 2,
                background: palette.primary.dark,
                color: palette.primary.contrastText,
            }}
        >
            <Typography
                component="h2" 
                variant="h4" 
                sx={{
                    width: '-webkit-fill-available',
                    textAlign: 'center',
                }}
            >
                {title}
                {helpText && <HelpButton markdown={helpText} /> }
            </Typography>
            <IconButton
                aria-label="close"
                edge="end"
                onClick={onClose}
            >
                <CloseIcon fill={palette.primary.contrastText} />
            </IconButton>
        </MuiDialogTitle>
    )
}