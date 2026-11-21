package analytics

// botUserAgentPatterns is deliberately conservative: only well-known
// automation, crawler, monitoring, and HTTP-client signatures are excluded.
var botUserAgentPatterns = []string{
	"bot", "crawler", "spider", "slurp", "headlesschrome", "phantomjs",
	"playwright", "puppeteer", "selenium", "lighthouse", "pingdom",
	"uptimerobot", "statuscake", "curl/", "wget/", "python-requests",
	"go-http-client", " okhttp", "libwww-perl", "scrapy", "facebookexternalhit",
	"semrush", "ahrefs", "bytespider", "petalbot", "bingpreview", "yandex",
}
