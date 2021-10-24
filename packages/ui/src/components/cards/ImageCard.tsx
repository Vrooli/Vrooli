import {
    Card,
    CardActions,
    CardContent,
    CardMedia,
    IconButton,
    Theme
} from '@material-ui/core';
import EditIcon from '@material-ui/icons/Edit';
import DeleteIcon from '@material-ui/icons/Delete';
import { makeStyles } from '@material-ui/styles';
import { useRef } from 'react';
import { useDrag, useDrop } from 'react-dnd';
import { combineStyles, getImageSrc } from 'utils';
import { IMAGE_SIZE, SERVER_URL } from '@local/shared';
import { cardStyles } from './styles';

const componentStyles = (theme: Theme) => ({
    imageContainer: {
        display: 'contents',
        position: 0,
    },
    displayImage: {
        height: 0,
        paddingTop: '56.25%',
    },
    actionButton: {
        color: theme.palette.secondary.light,
    },
})

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

interface Props {
    onDelete: () => any;
    onEdit: () => any;
    data: any;
    index: number;
    moveCard: (dragIndex: number, hoverIndex: number) => any;
}

export const ImageCard = ({
    onDelete,
    onEdit,
    data,
    index,
    moveCard
}: Props) => {
    const classes = useStyles();
    const ref = useRef(null);

    const [, drop] = useDrop({
        accept: 'card',
        collect(monitor) {
            return {
                handlerId: monitor.getHandlerId(),
            };
        },
        hover(item: any) {
            if (!ref.current) {
                return;
            }
            const dragIndex = item.index;
            const hoverIndex = index;
            // Don't replace items with themselves
            if (dragIndex === hoverIndex) {
                return;
            }
            moveCard(dragIndex, hoverIndex);
            item.index = hoverIndex;
        },
    });
    const [{ isDragging }, drag] = useDrag({
        type: 'card',
        item: () => {
            return { data, index };
        },
        collect: (monitor) => ({
            isDragging: monitor.isDragging(),
        }),
    });
    const opacity = isDragging ? 0 : 1;
    drag(drop(ref));

    return (
        <Card
            className={classes.cardRoot}
            style={{ opacity }}
            ref={ref}
        >
            <CardContent className={classes.imageContainer}>
                <CardMedia image={`${SERVER_URL}/${getImageSrc(data, IMAGE_SIZE.ML)}`} className={classes.displayImage} />
            </CardContent>
            <CardActions disableSpacing>
                <IconButton aria-label="edit image data" onClick={onEdit}>
                    <EditIcon className={classes.actionButton} />
                </IconButton>
                <IconButton aria-label="delete image" onClick={onDelete}>
                    <DeleteIcon className={classes.actionButton} />
                </IconButton>
            </CardActions>
        </Card>
    );
}