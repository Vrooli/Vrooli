import { Meta, Story } from "@storybook/react";
import { Dropzone as Component } from '../';
import { DropzoneProps as Props } from '../types';

// Define story metadata
export default {
    title: 'inputs/Dropzone',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});