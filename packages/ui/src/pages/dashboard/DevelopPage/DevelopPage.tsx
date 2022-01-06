import { makeStyles } from '@mui/styles';
import { Theme } from '@mui/material';
import { ResourceList } from 'components';

const useStyles = makeStyles((theme: Theme) => ({
    root: {}
}));

export const DevelopPage = ({

}) => {
    const classes = useStyles();

    return (
        <div id="page" className={classes.root}>
            asdfdsafdsafd
            asdf
            asdfdsafdsafddsa
            <ResourceList />
        </div>
    )
}