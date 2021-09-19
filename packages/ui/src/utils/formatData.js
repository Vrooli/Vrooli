// A collection of functions for parsing and serializing data from the backend
// Ex: phone numbers are often stored as a string of just numbers, 
// and formatted in the frontend to have spaces, parentheses, etc.
import _ from 'lodash';

// Convert price stored in database to user-friendly version
export const showPrice = (price) => {
    if (!price) return 'N/A';
    let result = Number(price);
    if (isNaN(result)) return price;
    return `$${result.toFixed(2)}`;
}

// Convert display price to database representation
export const storePrice = (price) => {
    // Convert to string, if needed
    let priceString = price + '';
    // Remove unit
    priceString = priceString.replace('$', '');
    let number = Number(priceString);
    // If number, return cents
    // If not a number, return 'N/A'
    return (_.isNumber(number) && number > 0) ? number.toFixed(2) : null;
}

// '15558675309' -> '+1 (555) 867-5309'
export const showPhone = (phone) => {
    var cleaned = ('' + phone).replace(/\D/g, '');
    var match = cleaned.match(/^(1|)?(\d{3})(\d{3})(\d{4})$/);
    if (match) {
        var intlCode = (match[1] ? '+1 ' : '');
        return [intlCode, '(', match[2], ') ', match[3], '-', match[4]].join('');
    }
    return null;
}

// '+1 (555) 867-5309' -> '15558675309'
export const storePhone = (phone) => {

}

// '15558675309' -> 'tel:+15558675309'
export const phoneLink = (phone) => `tel:${phone}`
export const emailLink = (address, subject = '', body = '') => `mailto:${address}?subject=${subject}&body=${body}`;