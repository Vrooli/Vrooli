import { useState } from 'react';
import { makeStyles } from '@material-ui/styles';
import Popover from '@material-ui/core/Popover';
import { Button, Theme } from '@material-ui/core';

const useStyles = makeStyles((theme: Theme) => ({
    paper: {
        background: theme.palette.primary.light,
    },
}));

export function PopupMenu({
    text = 'Menu',
    children,
    ...props
}) {
    const classes = useStyles();
    const [anchorEl, setAnchorEl] = useState(null);

    const handleClick = (event) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    const id = open ? 'simple-popover' : undefined;
    return (
        <>
            <Button aria-describedby={id} {...props} onClick={handleClick}>
                {text}
            </Button>
            <Popover
                id={id}
                open={open}
                anchorEl={anchorEl}
                onClose={handleClose}
                classes={{
                    paper: classes.paper
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
                {children}
            </Popover>
        </>
    )
}