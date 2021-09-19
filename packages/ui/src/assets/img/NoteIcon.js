import React from 'react';
import PropTypes from 'prop-types';

function NoteIcon(props) {
	return (
		<svg xmlns="http://www.w3.org/2000/svg" 
			className={props.className} 
			viewBox="0 0 512 512" 
			aria-labelledby="note-title" 
			width={props.width} 
			height={props.height}
			onClick={() => typeof props.onClick === 'function' && props.onClick()}>
		<title id="note-title">{props.iconTitle ?? 'Note'}</title>
        <path d="M352.459 220c0-11.046-8.954-20-20-20h-206c-11.046 0-20 8.954-20 20s8.954 20 20 20h206c11.046 0 20-8.954 20-20zM126.459 280c-11.046 0-20 8.954-20 20s8.954 20 20 20H251.57c11.046 0 20-8.954 20-20s-8.954-20-20-20H126.459z"/><path d="M173.459 472H106.57c-22.056 0-40-17.944-40-40V80c0-22.056 17.944-40 40-40h245.889c22.056 0 40 17.944 40 40v123c0 11.046 8.954 20 20 20s20-8.954 20-20V80c0-44.112-35.888-80-80-80H106.57c-44.112 0-80 35.888-80 80v352c0 44.112 35.888 80 80 80h66.889c11.046 0 20-8.954 20-20s-8.954-20-20-20z"/><path d="M467.884 289.572c-23.394-23.394-61.458-23.395-84.837-.016l-109.803 109.56c-2.332 2.327-4.052 5.193-5.01 8.345l-23.913 78.725c-2.12 6.98-.273 14.559 4.821 19.78 3.816 3.911 9 6.034 14.317 6.034 1.779 0 3.575-.238 5.338-.727l80.725-22.361c3.322-.92 6.35-2.683 8.79-5.119l109.573-109.367c23.394-23.394 23.394-61.458-.001-84.854zM333.776 451.768l-40.612 11.25 11.885-39.129 74.089-73.925 28.29 28.29-73.652 73.514zM439.615 346.13l-3.875 3.867-28.285-28.285 3.862-3.854c7.798-7.798 20.486-7.798 28.284 0 7.798 7.798 7.798 20.486.014 28.272zM332.459 120h-206c-11.046 0-20 8.954-20 20s8.954 20 20 20h206c11.046 0 20-8.954 20-20s-8.954-20-20-20z"/>		</svg>
	)
}

NoteIcon.propTypes = {
	iconTitle: PropTypes.string,
	className: PropTypes.string,
	onClick: PropTypes.func,
	width: PropTypes.string,
	height: PropTypes.string,
}

export { NoteIcon };