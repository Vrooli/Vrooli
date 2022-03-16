import { useState } from 'react';
import { makeStyles } from '@mui/styles';
import { Button, Popover, Theme } from '@mui/material';
import { PopupMenuProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    paper: {
        background: theme.palette.primary.light,
        borderRadius: '24px',
    },
}));

export function PopupMenu({
    text = 'Menu',
    children,
    ...props
}: PopupMenuProps) {
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
                disableScrollLock={true}
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