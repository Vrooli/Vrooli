/**
 * Helper script to convert fetchData tests from mockFetch pattern to MSW.
 * This script provides templates for common test patterns.
 */

// Common MSW patterns for fetchData tests:

// 1. Basic response mock:
`
server.use(
    http.{method}(\`\${apiUrlBase}\${restBase}{endpoint}\`, () => {
        return HttpResponse.json({ data: "response", version: "1.0.0" });
    })
);
`

// 2. Request capture pattern:
`
let capturedRequest: any = null;
server.use(
    http.{method}(\`\${apiUrlBase}\${restBase}{endpoint}\`, async ({ request }) => {
        capturedRequest = await request.json();
        return HttpResponse.json({ data: "response", version: "1.0.0" });
    })
);
`

// 3. Query parameters capture:
`
let capturedParams: any = null;
server.use(
    http.get(\`\${apiUrlBase}\${restBase}{endpoint}\`, ({ request }) => {
        const url = new URL(request.url);
        capturedParams = Object.fromEntries(url.searchParams.entries());
        return HttpResponse.json({ data: "response", version: "1.0.0" });
    })
);
`

// 4. Error response:
`
server.use(
    http.{method}(\`\${apiUrlBase}\${restBase}{endpoint}\`, () => {
        return HttpResponse.json(
            { errors: [{ message: "Test error" }] },
            { status: 500 }
        );
    })
);
`

// 5. Delayed response:
`
server.use(
    http.{method}(\`\${apiUrlBase}\${restBase}{endpoint}\`, async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
        return HttpResponse.json({ data: "delayed", version: "1.0.0" });
    })
);
`;

console.log("MSW conversion patterns loaded for fetchData tests");
