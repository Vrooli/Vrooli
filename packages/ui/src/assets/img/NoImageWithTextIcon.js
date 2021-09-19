import React from 'react';
import PropTypes from 'prop-types';

function NoImageWithTextIcon(props) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg"
            viewBox="12 12 73 90"
            className={props.className}
            aria-labelledby="noimagewithtext-title"
            width={props.width}
            height={props.height}
            onClick={() => typeof props.onClick === 'function' && props.onClick()}>
            <title id="noimagewithtext-title">{props.iconTitle ?? 'Image Not Available'}</title>
            <path d="M60 56v-7.6l-2-2 1.4-1.4.6.6v-1.2l2-2V57c0 .6-.4 1-1 1H46.4l2-2H60zm4 10H38.4l-2 2H65c.6 0 1-.4 1-1V38.4l-2 2V66zm9.7-38.3l-46 46a1 1 0 11-1.4-1.4l5.7-5.7V29c0-.6.4-1 1-1h32c.6 0 1 .4 1 1v3.6l6.3-6.3a1 1 0 111.4 1.4zM34 64.6l6.6-6.6H37a1 1 0 01-1-1V33c0-.6.4-1 1-1h24c.6 0 1 .4 1 1v3.6l2-2V30H34v34.6zM56.5 42l3.5-3.5V34H38v13.5l3.3-3.2a1 1 0 011.4 0l3.3 3.3 7.3-7.3a1 1 0 011.4 0l1.8 1.8zm-2.5.3l-7.3 7.3a1 1 0 01-1.4 0L42 46.4l-4 4V56h4.6L55 43.5l-1.1-1z" />
            <text x="25" y="80" fontSize="5" fontWeight="bold" fontFamily="'Helvetica Neue', Helvetica, Arial-Unicode, Arial, Sans-serif">
                No Image Available
            </text>
        </svg>
    )
}

NoImageWithTextIcon.propTypes = {
    iconTitle: PropTypes.string,
    className: PropTypes.string,
    onClick: PropTypes.func,
    width: PropTypes.string,
    height: PropTypes.string,
}

export { NoImageWithTextIcon };
