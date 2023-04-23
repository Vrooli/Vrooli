const openGraphRegex = /facebookexternalhit|twitterbot|linkedinbot|embedly|quora link preview|Pinterest|slackbot|vkShare|facebot/i;
export function isBot(userAgent) {
    console.log("checking user agent", userAgent);
    return openGraphRegex.test(userAgent);
}
//# sourceMappingURL=openGraph.js.map