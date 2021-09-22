import React from "react";
import PropTypes from "prop-types";

const useStyles = (theme) => ({
    videoResponsive: {
        overflow: 'hidden',
        paddingBottom: '56.25%',
        position: 'relative',
        height: 0,
        'iframe': {
            left: 0,
            top: 0,
            height: '100%',
            width: '100%',
            position: 'absolute',
        }
    }
})

function YoutubeEmbed({ 
    embedId, 
    width, 
    height 
}) {
    const classes = useStyles();

    return (
        <div className={classes.videoResponsive}>
            <iframe
                width={width}
                height={height}
                src={`https://www.youtube.com/embed/${embedId}`}
                frameBorder="0"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
                title="Embedded youtube"
            />
        </div>
    )
};

YoutubeEmbed.propTypes = {
    embedId: PropTypes.string.isRequired
};

export { YoutubeEmbed };