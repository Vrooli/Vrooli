import ical from "node-ical";
export const parseICalFile = async (file) => {
    const { createReadStream, filename, mimetype } = await file;
    const stream = createReadStream();
    const events = await ical.async.parseFile("example-calendar.ics");
};
//# sourceMappingURL=calendar.js.map