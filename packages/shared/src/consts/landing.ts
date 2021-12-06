import { ValueOf } from '.';

export const LANDING_LINKS = {
    About: '/about', // Overview of project, the vision, and the team
    Benefits: '/#understand-your-workflow', // Start of slides overviewing benefits of using Vrooli
    Home: '/', // Default page when not logged in. Similar to the about page, but more project details and less vision
    Mission: '/mission', // More details about the project's overall vision
    PrivacyPolicy: '/privacy-policy', // Privacy policy
    Roadmap: '/mission#roadmap', // Start of roadmap slide
    Terms: '/terms-and-conditions', // Terms and conditions
}
export type LANDING_LINKS = ValueOf<typeof LANDING_LINKS>;