# Testing UI Performance, Accessibility, and SEO
Performance, accessibility, and SEO are all important aspects of a website. They improve the user experience, and can help your website rank higher in search engines. This guide will show you how to test your website for these aspects, and how to improve them.

## Performance
Below are a few approaches to testing and improving performance.

**NOTE**: When testing for performance, make sure you are running a production build. This can be set with `NODE_ENV` in the .env file. If you would like to test performance locally, be mindful that certain performance features (such as cache policy) may be handled by Nginx, so they won't be available locally.

- [Lighthouse](https://developers.google.com/web/tools/lighthouse) is an open-source tool for testing any website's (even localhost) performance, accessibility, and Search Engine Optimization (SEO). This can be accessed in Chrome Dev Tools. The tool generates a report in less than a minute, which gives plenty of details and resources that you can look through. This website template is designed to maximize Lighthouse performance by default, but your specific needs may vary. Some places to look at are:  
- Compress static images - The easiest way to reduce request payloads is by compressing static images. This can be done on many sites, such as [this one for PNGs](https://compresspng.com/) and [this one](https://jakearchibald.github.io/svgomg/) for SVGs.
- Remove unused dependencies - The easiest way I've found to discover unused dependencies is with [depcheck](https://www.npmjs.com/package/depcheck):    
    1. In project's root directory, enter `yarn global add depcheck`  
    2. `depcheck` or `npx depcheck`  
    3. Repeat in each package (packages/server, packages/shared, packages/ui)  
Before removing packages, please make sure that depcheck was correct. If you are only using the package in a Dockerfile, for example, it may not catch it!
- Remove unused components and pages - Every byte counts with web responsiveness! One method for finding unused code is to use [ts-unused-exports](https://github.com/pzavolinsky/ts-unused-exports).
- Peek inside code bundles - Seeing what's inside the code bundles can help you determine what areas of the code should be lazy loaded, and what is taking the most space. To do this:  
    1. Make sure `sourcemap` is set to `true` in the `vite.config.ts` file.
    1. Navigate to the UI package: `cd packages/ui ` 
    2. `yarn build`
    3. `yarn analyze`
    4. Open the generated `source-tree.html` file in your browser. This will show you the size of each file in the bundle, and how they are related to each other.

## Accessibility
Accessibility is a crucial aspect of web development, ensuring that your website is usable by people with various disabilities. This section guides you on how to test and improve the accessibility of your website.

### Testing for Accessibility
1. **Automated Testing with Tools**: Use tools like [axe](https://www.deque.com/axe/) or Lighthouse (already mentioned in the Performance section) to perform automated accessibility checks. These tools can detect common accessibility issues such as missing alt text, poor contrast ratios, and incorrect ARIA attributes.
2. **Manual Testing**: While automated tools catch many issues, some require manual checking. This includes verifying keyboard navigation, ensuring all interactive elements are focusable and operable with a keyboard, and testing with screen readers like NVDA or VoiceOver.
3. **Browser Developer Tools**: Modern browsers have built-in accessibility inspectors (like Chrome's Accessibility Tree in DevTools) that can help you understand how assistive technologies interpret your page.
4. **Checklists and Guidelines**: Refer to the [Web Content Accessibility Guidelines (WCAG)](https://www.w3.org/WAI/standards-guidelines/wcag/) for a comprehensive set of guidelines. Use checklists like [WebAIM's WCAG Checklist](https://webaim.org/standards/wcag/checklist) for a more approachable format.

### Improving Accessibility
- **Semantic HTML**: Use semantic HTML elements (like `<button>`, `<nav>`, `<header>`, etc.) as they provide inherent accessibility features.
- **ARIA Attributes**: Use ARIA (Accessible Rich Internet Applications) attributes when necessary, but prefer native HTML solutions when possible.
- **Color Contrast**: Ensure that text and interactive elements have sufficient contrast against their backgrounds. Tools like [WebAIM's Contrast Checker](https://webaim.org/resources/contrastchecker/) can help.
- **Alt Text for Images**: Provide descriptive alt text for images, especially for informative images.
- **Labels for Interactive Elements**: Ensure all form inputs have associated labels, and interactive elements have accessible names.
- **Error Identification and Recovery**: Provide clear error messages and guidance for error correction in forms.
- **Responsive and Mobile Accessibility**: Ensure that your website is accessible on mobile devices, with touch-friendly interactive elements and readable text without zooming.
- **Captions and Transcripts**: For multimedia content, provide captions for videos and transcripts for audio content.
- **Testing with Real Users**: Conduct usability testing with people who have disabilities to get direct feedback on the accessibility of your site.

## SEO
- Sitemaps are used by search engines to index your website. This is especially important for client-side rendered websites, as search engines may not be able to crawl your website. We use [sitemap.ts](https://github.com/Vrooli/Vrooli/tree/master/packages/ui/src/__tools/sitemap/sitemap.ts) to do this automatically for static pages (e.g. /about, /auth/login), and [genSitemap.ts](https://github.com/Vrooli/Vrooli/tree/master/packages/jobs/src/schedules/genSitemap.ts) to do this for dynamic pages (e.g. /profile/:id). If these are working correctly, you shouldn't need to worry about sitemaps.
