import { Meta, Story } from "@storybook/react";
import { AlertDialog as Component } from '../';
import { AlertDialogProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/AlertDialog',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});