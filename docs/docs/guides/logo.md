# Logo
This section will guide you through the process of creating and applying a logo.

## Basic Design Principles
Creating a logo is an intricate task and understanding some basic design principles is crucial:

1. **Simplicity**: A simple logo design allows for easy recognition. Complex designs can be difficult to reproduce across different platforms and sizes.  
2. **Memorability**: An effective logo design should be memorable, and this is achieved by keeping it simple yet appropriate.  
3. **Versatility**: An effective logo should be versatile and be able to work across a variety of mediums and applications.   
4. **Appropriateness**: The logo should be appropriate for the business or product it represents.  

## Creating Different Logo Versions
For all devices and operating systems to display the logo correctly, it needs to be created in different sizes and formats. The basic workflow is as follows:
1. Create an SVG version of the logo, using a tool like [Inkscape](https://inkscape.org/).  
2. Create a square PNG version of the logo (512px x 512px), using a tool like [GIMP](https://www.gimp.org/). This is simply the logo in the center, with some solid or gradient background.  
3. Use a tool like [realfavicongenerator](https://realfavicongenerator.net/) to generate the different versions of the logo. Passing in the SVG (i.e. no background) often looks better for the Mac and Windows icons, and the PNG (i.e. with background) often looks better for the Android and iOS icons. You can also pass in a version of the PNG with rounded corners for Android.  
4. If your images are not in the `webp` format, you may want to use a tool to convert them. `webp` images are smaller and load faster than other formats.
5. Add the generated logo files to `packages/ui/public`. Also add the square PNG from step 2, and call it `logo-mask-512x512.webp`.

## Splash Screens
Splash screens are displayed on mobile devices while the app is loading. For Android, they are created automatically using the theme color and icon. For iOS, you need to create a splash screen for every device and theme combination. For this, you can use [Progressier](https://progressier.com/pwa-icons-and-ios-splash-screen-generator).  

## Manifest and index.html  
There are 3 places that need to define the logo: `packages/ui/index.html`, `packages/ui/public/manifest.light.json`, and `packages/ui/public/manifest.dark.json`.

Here's an example of what needs to be included in `packages/ui/index.html`:

```html
<link rel="apple-touch-startup-image"
        media="(prefers-color-scheme: light) and screen and (device-width: 430px) and (device-height: 932px) and (-webkit-device-pixel-ratio: 3) and (orientation: landscape)"
        href="/splash_screens/light/iPhone_14_Pro_Max_landscape.png">
<link rel="apple-touch-startup-image"
        media="(prefers-color-scheme: dark) and screen and (device-width: 430px) and (device-height: 932px) and (-webkit-device-pixel-ratio: 3) and (orientation: landscape)"
        href="/splash_screens/dark/iPhone_14_Pro_Max_landscape.png">
```

And here's an example of a manifest file that includes different icon sizes:

```json
{
  "name": "App Name",
  "short_name": "App",
  "icons": [
        {
            "src": "/favicon-16x16.png",
            "sizes": "16x16",
            "type": "image/png"
        },
        {
            "src": "/android-chrome-192x192.webp",
            "sizes": "192x192",
            "type": "image/webp"
        },
        {
            "src": "/android-chrome-512x512.webp",
            "sizes": "512x512",
            "type": "image/webp",
            "purpose": "any"
        },
        {
            "src": "/logo-mask-512x512.webp",
            "sizes": "512x512",
            "type": "image/webp",
            "purpose": "maskable"
        }
    ],
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#000000"
}
```
Some other important manifest fields to keep in mind are: 
1. `name` - Name of application in install dialog and Chrome Web Store. Maximum 45 characters  
2. `short_name` - Short version of application name. Maximum 12 characters  
3. `display` - either `fullscreen`, `standalone`, `minimal-ui`, or `browser`  
4. `scope` - usually `/`
4. `start_url` - usually `/`  
5. `background_color` - background color for splash screen and notifcations bar  

See [this web.dev guide](https://web.dev/add-manifest/) for more information on manifest fields.