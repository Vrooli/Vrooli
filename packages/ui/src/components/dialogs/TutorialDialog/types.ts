export type Place = {
    section: number;
    step: number;
}

// Simplified form structure types to avoid import issues
export type FormInformationalType = {
    type: string;
    id: string;
    label: string;
    tag?: string;
    color?: string;
    icon?: string;
    isCollapsible?: boolean;
    isMarkdown?: boolean;
}

export type TutorialStep = {
    /**
     * Action triggered when step is reached. 
     * Useful for opening side menus, putting components in 
     * a certain state, etc.
     */
    action?: () => unknown;
    content: FormInformationalType[];
    location?: {
        /**
         * ID of the element to anchor the tutorial step to.
         */
        element?: string;
        /**
         * What page to navigate to when the step is reached.
         */
        page?: string;
    };
    /**
     * Overrides the next step to go to.
     */
    next?: Place;
    /**
     * Allows you to specify options for what place to go next. 
     * Useful for giving the user a choice of what to learn about.
     * 
     * NOTE: If only one option is provided, the user will be taken there automatically. 
     * No option will be shown.
     */
    options?: {
        label: string;
        place: Place;
    }[];
    /**
     * Overrides the previous step.
     */
    previous?: Place;
}

export type TutorialSection = {
    hideFromMenu?: boolean;
    title: string;
    steps: TutorialStep[];
}

export type TutorialDialogProps = {
    bypassPageValidation?: boolean;
};
