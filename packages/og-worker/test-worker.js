#!/usr/bin/env node

/**
 * Simple test script for the OG Worker without needing Wrangler dev
 * Tests key functionality: bot detection, caching logic, health checks
 */

console.log("🧪 Testing OG Worker functionality...\n");

// Test 1: Bot Detection
console.log("1️⃣ Testing Bot Detection:");
const botAgents = [
    "Googlebot/2.1",
    "facebookexternalhit/1.1",
    "Twitterbot/1.0",
    "Mozilla/5.0 (compatible; bingbot/2.0)",
    "Slackbot-LinkExpanding 1.0",
    "WhatsApp/2.19.81",
];

const humanAgents = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X)",
    "curl/7.68.0",
];

const BOT_UA_STRING =
    "(googlebot|google-structured-data-testing-tool|google-inspectiontool|bingbot|" +
    "bingpreview|slurp|duckduckbot|baiduspider|yandexbot|facebookexternalhit|facebot|" +
    "facebookcatalog|meta-external(?:agent|fetcher)|twitterbot|pinterest(?:bot)?|" +
    "linkedinbot|slackbot-linkexpanding|discordbot|telegrambot|whatsapp|" +
    "skypeuripreview|embedly|opengraph|iframely|vkshare|bitlybot|" +
    "applebot|chrome-lighthouse|pagespeed|gtmetrix|pingdom|uptimerobot|" +
    "statuscake|newrelic|datadog|semrush|ahrefs|mj12bot)";
const BOT_UA = new RegExp(BOT_UA_STRING, "i");

console.log("Bot agents (should return true):");
botAgents.forEach(agent => {
    const isBot = BOT_UA.test(agent);
    console.log(`  ${isBot ? "✅" : "❌"} ${agent}`);
});

console.log("\nHuman agents (should return false):");
humanAgents.forEach(agent => {
    const isBot = BOT_UA.test(agent);
    console.log(`  ${!isBot ? "✅" : "❌"} ${agent}`);
});

// Test 2: Health Check Paths
console.log("\n2️⃣ Testing Health Check Paths:");
const HEALTH_CHECK_PATHS = ["/health", "/api/health", "/healthz", "/ready", "/live"];
const testPaths = [
    "/health",
    "/api/health",
    "/healthz",
    "/ready",
    "/live",
    "/api/v2/user/123",
    "/about",
];

testPaths.forEach(path => {
    const isHealthCheck = HEALTH_CHECK_PATHS.includes(path);
    console.log(`  ${isHealthCheck ? "🏥" : "📄"} ${path} - ${isHealthCheck ? "Health check" : "Normal route"}`);
});

// Test 3: Cache Key Generation
console.log("\n3️⃣ Testing Cache Key Generation:");
const testUrls = [
    "https://vrooli.com/user/123",
    "https://vrooli.com/user/123?utm_source=twitter",
    "https://vrooli.com/user/123#section",
    "https://app.vrooli.com/api/health",
];

testUrls.forEach(url => {
    const urlObj = new URL(url);
    const cacheUrl = `${urlObj.protocol}//${urlObj.host}${urlObj.pathname}`;
    console.log(`  ${url}`);
    console.log(`    → Cache key: ${cacheUrl}`);
});

// Test 4: Escape Function
console.log("\n4️⃣ Testing HTML Escape Function:");
function esc(s) {
    const escapeMap = {
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "\"": "&quot;",
        "'": "&#39;",
        "`": "&#96;"
    };
    return s ? s.replace(/[&<>"'`]/g, (c) => escapeMap[c] || c) : "";
}

const testStrings = [
    'Hello <script>alert("XSS")</script>',
    "O'Reilly & Associates",
    'Link: <a href="test">click</a>',
    "Backtick: `code`",
    null,
    undefined,
    "",
];

testStrings.forEach(str => {
    const escaped = esc(str);
    console.log(`  "${str}" → "${escaped}"`);
});

console.log("\n✅ All tests completed!");
console.log("\n📝 To run the actual worker locally with Node 20:");
console.log("  1. Use Docker: docker compose -f docker-compose.test.yml up");
console.log("  2. Or use nvm: nvm use 20 && npx wrangler dev --local");