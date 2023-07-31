# PWAs and TWAs

## Progressive Web App (PWA)
A PWA is a website that can be installed on mobile devices. These don't have quite the same functionality as native apps, but hopefully one day they will. To make your website PWA-compatable, perform an audit on Lighthouse (found in your browser's developer console). Then, follow the steps it provides. **Make sure `NODE_ENV` is set to `production` when testing PWA.**

## Trusted Web Activity (TWA)
A trusted web activity is a PWA that runs natively on Android devices. They can also be listed on the Google Play store, making them almost identical to traditional apps. If this sounds interesting to you, make sure that the `packages/ui/public/site.manifest` or `packages/ui/public/manifest.json` file has the following data:  
1. orientation and display to define how the app should feel (likely "any" and "standalone" to feel like a native app)
2. screenshots (displayed in the store)  
3. name, short name, and description   
4. dir (direction of text) and lang for localization
5. icons
6. start_url and scope
, along with everything else mentioned in the Favicons and PWA sections of this guide. All known manifest fields can be found [here](https://developer.mozilla.org/en-US/docs/Web/Manifest/categories).

Once that is complete, you can use the [PWABuilder tool](https://www.pwabuilder.com/) to generate other required files and receive further instructions.

When you're ready to test or deploy your app to the Google Play store, visit [Google Play Console](https://play.google.com/) to receive instructions.