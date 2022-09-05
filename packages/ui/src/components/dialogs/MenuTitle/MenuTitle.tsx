import {
    Box,
    IconButton,
    Typography,
    useTheme
} from '@mui/material';
import { CloseIcon } from '@shared/icons';
import { HelpButton } from 'components';
import { noSelect } from 'styles';
import { MenuTitleProps } from '../types';

export const MenuTitle = ({
    ariaLabel,
    helpText,
    onClose,
    title,
}: MenuTitleProps) => {
    const { palette } = useTheme();

    return (
        <Box
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
                variant="h5" 
                sx={{
                    width: '-webkit-fill-available',
                    textAlign: 'center',
                }}
            >
                {title}
                {helpText && <HelpButton markdown={helpText} sx={{ fill: palette.secondary.light }} /> }
            </Typography>
            <IconButton
                aria-label="close"
                edge="end"
                onClick={onClose}
            >
                <CloseIcon fill={palette.primary.contrastText} />
            </IconButton>
        </Box>
    )
}