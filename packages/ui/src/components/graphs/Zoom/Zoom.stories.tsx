import { Meta, Story } from "@storybook/react";
import { ZoomI as Component } from '../';
import { ZoomIProps as Props } from '../types';

// Define story metadata
export default {
    title: 'graphs/ZoomI',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});