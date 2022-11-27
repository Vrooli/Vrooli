// TODO might not work if ID appears multiple times in data, where the first
// result is not the one we want
/**
 * Picks an object from a nested object, using the given ID
 * @param data Object array to pick from
 * @param id ID to pick
 * @returns Requested object with all its fields and children included. If object not found, 
 * returns { id }
 */
export const pickObjectById = (data: any, id: string): ({ id: string } & { [x: string]: any }) => {
    // Stringify data, so we can perform search of ID
    const dataString = JSON.stringify(data);
    // Find the location in the string where the ID is. 
    // Data is only found if there are more fields than just the ID
    const searchString = `"id":"${id}",`;
    const idIndex = dataString.indexOf(searchString);
    // If ID not found
    if (idIndex === -1) return { id };
    // Loop backwards until we find the start of the object (i.e. first unmatched open bracket before ID)
    let openBracketCounter = 0;
    let inQuotes = false;
    let startIndex = idIndex - 1;
    let lastChar = dataString[idIndex];
    while (startIndex >= 0) {
        const currChar = dataString[startIndex];
        // If current and last char are "\", then the next character is escaped and should be ignored
        if (currChar !== '\\' && lastChar !== '\\') {
            // Don't count bracket if it appears in quotes (i.e. part of a string)
            if (!inQuotes) {
                if (dataString[startIndex] === '{') openBracketCounter++;
                else if (dataString[startIndex] === '}') openBracketCounter--;
                // If we found the closing bracket, we're done
                if (openBracketCounter === 1) {
                    break;
                }
            }
            else if (dataString[startIndex] === '"') inQuotes = !inQuotes;
        } else startIndex--;
        lastChar = dataString[startIndex];
        startIndex--;
    }
    // If start is not found
    if (startIndex === -1) return { id };
    // Loop forwards through string until we find the end of the object
    openBracketCounter = 1;
    inQuotes = false;
    let endIndex = idIndex + searchString.length;
    lastChar = dataString[idIndex + searchString.length];
    while (endIndex < dataString.length) {
        const currChar = dataString[endIndex];
        // If current and last char are "\", then the next character is escaped and should be ignored
        if (currChar !== '\\' && lastChar !== '\\') {
            // Don't count bracket if it appears in quotes (i.e. part of a string)
            if (!inQuotes) {
                if (dataString[endIndex] === '{') openBracketCounter++;
                else if (dataString[endIndex] === '}') openBracketCounter--;
                // If we found the closing bracket, we're done
                if (openBracketCounter === 0) {
                    break;
                }
            }
            else if (dataString[endIndex] === '"') inQuotes = !inQuotes;
        } else endIndex++;
        lastChar = dataString[endIndex];
        endIndex++;
    }
    // If end is not found, return undefined
    if (endIndex === dataString.length) return { id };
    // Return object
    return JSON.parse(dataString.substring(startIndex, endIndex + 1));
}
