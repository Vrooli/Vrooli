import { useCallback, useMemo, useState } from 'react';
import update from 'immutability-helper';
import { makeStyles } from '@material-ui/styles';
import { ImageCard } from 'components';
import { EditImageDialog } from 'components';

const useStyles = makeStyles(() => ({
    flexed: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
        alignItems: 'stretch',
    },
}));

interface Props {
    data: any[];
    onUpdate: (data: any) => any;
}

export const ImageList = ({
    data,
    onUpdate
}: Props) => {
    const classes = useStyles();
    const [selected, setSelected] = useState(-1);

    const moveCard = useCallback((dragIndex, hoverIndex) => {
        const dragCard = data[dragIndex];
        onUpdate(update(data, {
            $splice: [
                [dragIndex, 1],
                [hoverIndex, 0, dragCard],
            ],
        }));
    }, [data, onUpdate]);

    const saveImageData = useCallback((d) => {
        let updated = [...data];
        updated[selected] = {
            ...updated[selected],
            ...d
        };
        onUpdate(updated);
        setSelected(-1);
    }, [selected, data, onUpdate])

    const deleteImage = useCallback((index) => {
        let updated = [...data];
        updated.splice(index, 1);
        onUpdate(updated);
    }, [data, onUpdate]);

    const onDialogClose = useCallback(() => setSelected(-1), []);

    const imageCards = useMemo(() => (
        data?.map((item, index) => (
            <ImageCard
                key={index}
                index={index}
                data={item}
                onDelete={() => deleteImage(index)}
                onEdit={() => setSelected(index)}
                moveCard={moveCard}
            />
        ))
    ), [data, deleteImage, moveCard]);

    return (
        <div className={classes.flexed}>
            <EditImageDialog
                open={selected >= 0}
                data={selected >= 0 ? data[selected] : null}
                onClose={onDialogClose}
                onSave={saveImageData}
            />
            {imageCards}
        </div>
    );
}