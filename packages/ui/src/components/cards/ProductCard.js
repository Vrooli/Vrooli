import React from 'react';
import PropTypes from 'prop-types'
import {
    Card,
    CardActionArea,
    CardActions,
    CardContent,
    CardMedia,
    IconButton,
    Tooltip,
    Typography
} from '@material-ui/core';
import { Launch as LaunchIcon } from '@material-ui/icons';
import { makeStyles } from '@material-ui/styles';
import { combineStyles, getImageSrc } from 'utils';
import { NoImageWithTextIcon } from 'assets/img';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { cardStyles } from './styles';

const componentStyles = (theme) => ({
    displayImage: {
        minHeight: 200,
        maxHeight: '50%',
        position: 'absolute',
    },
    topMargin: {
        marginTop: 200,
    },
    deleted: {
        border: '2px solid red',
    },
    inactive: {
        border: '2px solid grey',
    },
    active: {
        border: `2px solid ${theme.palette.secondary.dark}`,
    },
})

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

function ProductCard({
    onClick,
    product,
}) {
    const classes = useStyles();

    let display;
    let display_data = product.images.find(image => image.usedFor === IMAGE_USE.ProductDisplay)?.image;
    if (!display_data && product.images.length > 0) display_data = product.images[0].image;
    if (display_data) {
        display = <CardMedia component="img" src={`${SERVER_URL}/${getImageSrc(display_data)}`} className={classes.displayImage} alt={display_data.alt} title={product.name} />
    } else {
        display = <NoImageWithTextIcon className={classes.displayImage} />
    }

    return (
        <Card className={classes.cardRoot} onClick={() => onClick({ product, selectedSku: product.skus[0] })}>
            <CardActionArea>
                {display}
                <CardContent className={`${classes.content} ${classes.topMargin}`}>
                    <Typography gutterBottom variant="h6" component="h3">
                        {product.name}
                    </Typography>
                </CardContent>
            </CardActionArea>
            <CardActions>
                <Tooltip title="View" placement="bottom">
                    <IconButton onClick={onClick}>
                        <LaunchIcon className={classes.icon} />
                    </IconButton>
                </Tooltip>
            </CardActions>
        </Card>
    );
}

ProductCard.propTypes = {
    onClick: PropTypes.func.isRequired,
    product: PropTypes.object.isRequired,
}

export { ProductCard };