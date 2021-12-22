// Wraps Organization, Actor, Project, or Routine page in a dialog.
// Used if components were navigated to, rather than directly loaded via url

import { makeStyles } from '@material-ui/styles';
import {
    AppBar,
    Button,
    Dialog,
    Grid,
    IconButton,
    Theme,
    Toolbar,
    Typography,
} from '@material-ui/core';
import {
    Close as CloseIcon,
} from '@material-ui/icons';
import { ComponentWrapperDialogProps } from '../types';
import { UpTransition } from '..';

const useStyles = makeStyles((theme: Theme) => ({
    appBar: {
        position: 'relative',
    },
    title: {
        textAlign: 'center',
    },
    optionsContainer: {
        padding: theme.spacing(2),
        background: theme.palette.primary.main,
    },
    container: {
        background: theme.palette.background.default,
        flex: 'auto',
        padding: theme.spacing(1),
        paddingBottom: '15vh',
    },
    bottom: {
        background: theme.palette.primary.main,
        position: 'fixed',
        bottom: '0',
        width: '-webkit-fill-available',
        zIndex: 1,
    },
}));

export const ComponentWrapperDialog = ({

}: ComponentWrapperDialogProps) => {
    const classes = useStyles();

    return (
        <Dialog fullScreen open={true} onClose={() => {}} TransitionComponent={UpTransition}>
            <AppBar className={classes.appBar}>
                <Toolbar>
                    <IconButton edge="start" color="inherit" onClick={() => {}} aria-label="close">
                        <CloseIcon />
                    </IconButton>
                    <Grid container spacing={0}>
                        <Grid className={classes.title} item xs={12}>
                            <Typography variant="h5">
                                boop
                            </Typography>
                            <Typography variant="h6">
                                beep
                            </Typography>
                        </Grid>
                    </Grid>
                </Toolbar>
            </AppBar>
            <div className={classes.container}>
    
                <div className={classes.bottom}>
                    
                </div>
            </div>
        </Dialog>
    );
}