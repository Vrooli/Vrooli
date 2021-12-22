import { Meta, Story } from "@storybook/react";
import { ComponentWrapperDialog as Component } from '../';
import { ComponentWrapperDialogProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/ComponentWrapperDialog',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});