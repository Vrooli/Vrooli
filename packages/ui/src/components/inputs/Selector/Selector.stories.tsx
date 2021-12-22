import { Meta, Story } from "@storybook/react";
import { Selector as Component } from '../';
import { SelectorProps as Props } from '../types';

// Define story metadata
export default {
    title: 'inputs/Selector',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});