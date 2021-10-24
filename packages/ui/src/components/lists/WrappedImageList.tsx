import { useCallback, useEffect, useState } from 'react';
import { makeStyles } from '@material-ui/styles';
import { Button, Grid, Theme } from '@material-ui/core';
import { ImageList } from './ImageList';

const useStyles = makeStyles((theme: Theme) => ({
    pad: {
        marginBottom: theme.spacing(2),
        marginTop: theme.spacing(2)
    },
    gridItem: {
        display: 'flex',
    },
}));

interface Props {
    data: any[];
    onApply: (data: any) => any;
}

export const WrappedImageList = ({
    data,
    onApply
}: Props) => {
    const classes = useStyles();
    const [changed, setChanged] = useState<any>(null);

    useEffect(() => {
        setChanged(data);
    }, [data])

    const onApplyClick = useCallback(() => onApply(changed), [changed, onApply]);
    const onRevertClick = useCallback(() => setChanged(data), [data]);
    const onUpdate = useCallback((d) => setChanged(d), []);

    let options = (
        <Grid classes={{ container: classes.pad }} container spacing={2}>
            <Grid className={classes.gridItem} item xs={12} sm={6}>
                <Button fullWidth onClick={onApplyClick}>Apply Changes</Button>
            </Grid>
            <Grid className={classes.gridItem} item xs={12} sm={6}>
                <Button fullWidth onClick={onRevertClick}>Revert Changes</Button>
            </Grid>
        </Grid>
    )

    return (
        <>
            { options}
                <ImageList data={changed} onUpdate={onUpdate}/>
            { options}
        </>
    );
}