import { Meta, Story } from "@storybook/react";
import { HelpButton, Props } from './HelpButton';

// Define story metadata
export default {
    title: 'HelpButton',
    component: HelpButton,
    args: {
        title: 'Random title',
        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.',
    }
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <HelpButton {...args} />;

// Export story
export const Default = Template.bind({});