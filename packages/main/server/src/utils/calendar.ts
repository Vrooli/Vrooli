import ical from "node-ical";

export const parseICalFile = async (file: any): Promise<any> => {
    const { createReadStream, filename, mimetype } = await file;
    const stream = createReadStream();
    const events = await ical.async.parseFile("example-calendar.ics");
    // TODO: Parse
};
