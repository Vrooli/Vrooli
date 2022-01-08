import { Link } from 'wouter';
import { Button } from '@mui/material';
import { makeStyles } from '@mui/styles';
import { combineStyles } from 'utils';
import { APP_LINKS } from '@local/shared';
import { pageStyles } from './styles';

const componentStyles = () => ({
    center: {
        position: 'absolute',
        top: '50%',
        left: '50%',
        transform: 'translateX(-50%) translateY(-50%)',
    },
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

export const NotFoundPage = () => {
    const classes = useStyles();

    return (
        <div id="page">
            <div className={classes.center}>
                <h1>Page Not Found</h1>
                <h3>Looks like you've followed a broken link or entered a URL that doesn't exist on this site</h3>
                <br />
                <Link to={APP_LINKS.Home}>
                    <Button>Go to Home</Button>
                </Link>
            </div>
        </div>
    );
}