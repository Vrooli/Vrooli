import React from "react";
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles((theme) => ({
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
}));

interface Props {
    className?: string;
    embedId: string;
    width?: number;
    height?: number;
}

export const YoutubeEmbed = ({ 
    className,
    embedId, 
    width, 
    height 
}: Props) => {
    const classes = useStyles();

    return (
        <div className={`${classes.videoResponsive} ${className}`}>
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