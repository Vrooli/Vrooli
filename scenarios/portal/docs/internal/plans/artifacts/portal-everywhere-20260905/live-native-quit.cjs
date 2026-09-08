// CDP routing did not intercept this Electron session. Use the
// fetch fixture, which proves interception before submitting a message.
const fs = require('node:fs');
fs.copyFileSync(__dirname + '/native-companion-live.json', __dirname + '/native-quit-fetch-live.json');
require('./live-native-quit-fetch.cjs');
