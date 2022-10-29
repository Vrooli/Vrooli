// Express server that hosts the UI. 
// This is used so we can change meta tags before the page is rendered.
// In the future, this can also be used for precaching (where you hydrate the page with HTML before the JS loads).

const express = require('express');
const path = require('path');
const fs = require('fs');

const PORT = process.env.PORT_UI || 3000;

const openGraphRegex = /facebookexternalhit|twitterbot|linkedinbot|embedly|quora link preview|Pinterest|slackbot|vkShare|facebot/i;

const app = express();

// app.use(express.static(path.resolve(__dirname, './build')));

app.use(express.static(path.join(__dirname, "./build")));
app.use(express.static("./public"));

app.use((req, res, next) => {
    console.log('in get')
    const isBot = openGraphRegex.test(req.headers['user-agent']);
    console.log(`User-Agent: ${req.headers['user-agent']}. Is bot: ${isBot}`);
    res.sendFile(path.join(__dirname, "./build", "index.html"));
});

// app.get('/*', (_req, res) => {
//     console.log('in get')
//     const isBot = openGraphRegex.test(_req.headers['user-agent']);
//     console.log(`User-Agent: ${_req.headers['user-agent']}. Is bot: ${isBot}`);
//     const filePath = path.resolve(__dirname, './build', 'index.html');
//     fs.readFile(filePath, 'utf8', (err, htmlData) => {
//         if (err) {
//             console.error('read err', err);
//             return res.status(404).end();
//         }

//         return res.send(
//             htmlData.replace(/__TEST__/g, 'Hello World')
//         );
//     });
// })

// app.use(express.static(path.resolve(__dirname, './build')));

app.listen(PORT, () => {
    console.log(`UI Server listening on port ${PORT}`);
})