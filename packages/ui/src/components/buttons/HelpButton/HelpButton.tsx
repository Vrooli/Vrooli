import { useMemo, useState } from 'react';
import { makeStyles } from '@mui/styles';
import { IconButton, Popover, Theme, Tooltip } from '@mui/material';
import { HelpOutline as HelpIcon } from "@mui/icons-material";
import { HelpButtonProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        display: 'inline',
    },
    button: {
        display: 'inline-flex',
        bottom: '2px',
        fill: theme.palette.secondary.contrastText,
    },
    popupPaper: {
        background: theme.palette.primary.light,
    },
}));

export const HelpButton = ({
    title = '', // What the tooltip displays
    description,
    id = 'help-details-popover'
}: HelpButtonProps) => {
    const classes = useStyles();
    const [anchorEl, setAnchorEl] = useState(null);

    const openPopup = (event) => setAnchorEl(event.currentTarget);
    const closePopup = () => setAnchorEl(null);

    const popup_open = Boolean(anchorEl);
    const popup_id = popup_open ? id : undefined;

    const popup = useMemo(() => {
        //Todo design popup
        return (<div>{description}</div>)
    }, [description])

    return (
        <div className={classes.root}>
            <Tooltip placement="top" title={title}>
                <IconButton className={classes.button}>
                    <HelpIcon onClick={openPopup} />
                    <Popover
                        id={popup_id}
                        open={popup_open}
                        disableScrollLock={true}
                        anchorEl={anchorEl}
                        onClose={closePopup}
                        classes={{
                            paper: classes.popupPaper
                        }}
                        anchorOrigin={{
                            vertical: 'bottom',
                            horizontal: 'center',
                        }}
                        transformOrigin={{
                            vertical: 'top',
                            horizontal: 'center',
                        }}
                    >
                        {popup}
                    </Popover>
                </IconButton>
            </Tooltip>
        </div>
    )
}