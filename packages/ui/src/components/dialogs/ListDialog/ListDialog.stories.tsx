import { Meta, Story } from "@storybook/react";
import { ListDialog as Component } from '../';
import { ListDialogProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/ListDialog',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});