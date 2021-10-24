# Setting business fields
[assets/public/business.json](assets/public/business.json) contains business fields - such as the name, phone number, and address. Storing this information in the backend allows users with an Admin or Owner role to update them without having to reload the website or mess with the server (there isn't a page that does this in this repo, but it would be similar to how [packages/ui/src/pages/admin/AdminContactPage/AdminContactPage.ts](packages/ui/src/pages/admin/AdminContactPage/AdminContactPage.ts) works).

You can find some examples of these fields being used in:  
- [packages/server/src/worker/email/queue.ts](packages/server/src/worker/email/queue.ts)
- [packages/ui/src/App.ts](packages/ui/src/App.ts)
- [packages/ui/src/pages/PrivacyPolicyPage/PrivacyPolicyPage.ts](packages/ui/src/pages/PrivacyPolicyPage/PrivacyPolicyPage.ts)

You can create any fields you want in this file, but here's an example:  
```
{
    "BUSINESS_NAME": {
        "Short": "Dunder Mifflin", // Name displayed in Navbar
        "Long": "Dunder Mifflin Inc." // Name
    },
    "ADDRESS": {
        "Label": "123 Not A Real Road Scranton, PA 18505", // Address displayed in buttons, labels, tooltips, etc.
        "Link": "https://www.google.com/maps/place/Scranton,+PA/@41.4163608,-75.6630369,21z/data=!4m5!3m4!1s0x89c4d93a77484bbb:0xfff27920ab9bfae8!8m2!3d41.408969!4d-75.6624122" // Link to address. Can be found by clicking your business's address in Google Maps, then copying the url
    },
    "PHONE": {
        "Label": "(202) 555-0138",
        "Link": "tel:+12025550138" // Link that automatically opens phone app when clicked
    },
    "EMAIL": {
        "Label": "info@dundermifflin.fakesite",
        "Link": "mailto:info@dundermifflin.fakesite" // Link that automatically opens up a new email to this address. This can be appended with other data, such as a subject, CCs, and a body. See https://css-tricks.com/snippets/html/mailto-links/ for examples
    },
    "SOCIAL": {
        "Facebook": "https://www.facebook.com/notarealfacebooklink",
        "Instagram": "https://www.instagram.com/notarealinstagramlink"
    },
    "WEBSITE": "https://yourwebsiteurl.fakesite" // For links that redirect to the website. Might seem a little pointless, but is useful for the privacy policy page.
}
```
Note that json files cannot have comments, so they must be removed if you are copying this snippet.