import { Meta, Story } from "@storybook/react";
import { DecisionNode as Component } from '../';
import { DecisionNodeProps as Props } from '../types';

// Define story metadata
export default {
    title: 'nodes/DecisionNode',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});