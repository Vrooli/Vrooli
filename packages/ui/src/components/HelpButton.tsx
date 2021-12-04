import { useMemo, useState } from 'react';
import { makeStyles } from '@material-ui/styles';
import { IconButton, Popover, Theme, Tooltip } from '@material-ui/core';
import { HelpOutline as HelpIcon } from "@material-ui/icons";

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

interface Props {
    title?: string;
    description?: string;
    id?: string;
}

export const HelpButton = ({
    title = '', // What the tooltip displays
    description,
    id = 'help-details-popover'
}: Props) => {
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