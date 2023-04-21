/**
 * Regex to test if a user agent is an open graph crawler. Make sure 
 * not to include googlebot, baiduspider, or any other search engine
 * crawlers here.
 */
const openGraphRegex = /facebookexternalhit|twitterbot|linkedinbot|embedly|quora link preview|Pinterest|slackbot|vkShare|facebot/i;

/**
 * Determines if a user agent is a bot (sprcifically, an open graph crawler) or not.
 * @param req The request object
 * @returns True if the user agent is a bot, false otherwise
 */
export function isBot(userAgent: string): boolean {
    console.log('checking user agent', userAgent);
    return openGraphRegex.test(userAgent);
}