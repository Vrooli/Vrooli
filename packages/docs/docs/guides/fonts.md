# Fonts
It's a good idea to use custom fonts for the UI because it makes the app look more professional and unique. As of now (v1.9.6), the UI uses the [Roboto](https://fonts.google.com/specimen/Roboto) font for body text and [Open Sans](https://fonts.google.com/specimen/Open+Sans) for headings.

The easiest way to find custom fonts is by using [Google Fonts](https://fonts.google.com/). After selecting your desired fonts, there should be an option to view *selected families*. From here, you can copy the `<link>` tags and paste it into the UI's index.html file. Then, you can use the font in your components like so:

```javascript
    navName: {
        ...
        fontSize: '3.5em',
        fontFamily: `Roboto`,
    },
```

Alternatively, you can supply your own fonts. Using a site such as [1001 Fonts](https://www.1001fonts.com/) allows you to download a `.woff` or `.woff2` file for your desired font. This can be placed in the UI's `assets` folder, and registered in the global css section of your `App.tsx` like so:  

```javascript
    import Roboto from "./assets/fonts/Roboto.woff";
    ...
    "@global": {
        ...
        '@font-face': {
            fontFamily: 'Roboto',
            src: `local('Roboto'), url(${Roboto}) format('truetype')`,
            fontDisplay: 'swap',
        }
    },
```

Then, when you need to use the font, you can reference it the same way as the `index.html` method above.

When supplying a font, it is a good idea to compress it using [Font Squirrel](https://www.fontsquirrel.com/tools/webfont-generator). If you know which characters you need (such as for a logo), you can also delete unneeded characters via an app like [FontForge](https://fontforge.org/). In web development, size matters😉  

If you want to go even further (though it is probably not necessary), you can also encode your font as a base64 string so it can be used without fetching. On a Unix-based terminal, a `.woff` can be converted to base64 using the command `base64 -w 0 yourfont.woff > yourfont-64.txt`. Then, you can enter that string into the `@font-face` src like so: 

```javascript
    ...
    '@font-face': {
        ...
        src: `local('SakBunderan'), url(data:font/woff;charset=utf-8;base64,insertyourbase64stringhere) format('truetype')`,
        ...
    }
```
