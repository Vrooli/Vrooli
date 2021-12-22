import { Meta, Story } from "@storybook/react";
import { NewProjectDialog as Component } from '../';
import { NewProjectDialogProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/NewProjectDialog',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});