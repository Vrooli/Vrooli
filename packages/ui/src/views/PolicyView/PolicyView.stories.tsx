import { PrivacyPolicyView, TermsView } from "./PolicyView.js";

export default {
    title: "Views/PolicyView",
    component: PrivacyPolicyView,
};

export function PrivacyPolicy() {
    return (
        <PrivacyPolicyView />
    );
}

export function Terms() {
    return (
        <TermsView />
    );
}
